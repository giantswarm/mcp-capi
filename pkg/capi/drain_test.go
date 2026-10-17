package capi

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

const (
	drainNode = "worker-1"
	webPod    = "apps/web"
)

// drainPod returns a pod on node, controlled by a controller of kind (none
// when kind is empty).
func drainPod(name, node, kind string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "apps", Name: name, UID: types.UID(name)},
		Spec:       corev1.PodSpec{NodeName: node},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	if kind != "" {
		pod.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: appsv1.SchemeGroupVersion.String(), Kind: kind, Name: name + "-owner", Controller: new(true),
		}}
	}
	return pod
}

// drainClient returns a Client over a fake clientset holding drainNode and
// pods, whose evictions delete the pod unless refusals[pod name] is still
// positive, in which case the eviction is refused like a
// PodDisruptionBudget does and the count goes down. A negative count refuses
// for ever.
func drainClient(t *testing.T, policy WritePolicy, refusals map[string]int, pods ...*corev1.Pod) (*Client, *fake.Clientset) {
	t.Helper()
	previous := drainPollInterval
	drainPollInterval = time.Millisecond
	t.Cleanup(func() { drainPollInterval = previous })

	objects := []runtime.Object{&corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: drainNode}}}
	for _, pod := range pods {
		objects = append(objects, pod)
	}
	clientset := fake.NewClientset(objects...)
	clientset.PrependReactor("create", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		create, ok := action.(k8stesting.CreateAction)
		if !ok || create.GetSubresource() != "eviction" {
			return false, nil, nil
		}
		eviction := create.GetObject().(*policyv1.Eviction)
		if n := refusals[eviction.Name]; n != 0 {
			refusals[eviction.Name] = n - 1
			return true, nil, apierrors.NewTooManyRequests("Cannot evict pod as it would violate the pod's disruption budget.", 0)
		}
		gvr := corev1.SchemeGroupVersion.WithResource("pods")
		return true, nil, clientset.Tracker().Delete(gvr, eviction.Namespace, eviction.Name)
	})
	c := &Client{}
	WithWritePolicy(policy)(c)
	c.SetClients(clientset, nil)
	return c, clientset
}

func evictions(clientset *fake.Clientset) []string {
	var names []string
	for _, action := range clientset.Actions() {
		if create, ok := action.(k8stesting.CreateAction); ok && create.GetSubresource() == "eviction" {
			names = append(names, create.GetObject().(*policyv1.Eviction).Name)
		}
	}
	return names
}

func podExists(t *testing.T, clientset *fake.Clientset, name string) bool {
	t.Helper()
	_, err := clientset.CoreV1().Pods("apps").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("get pod %s: %v", name, err)
	}
	return err == nil
}

func assertCordoned(t *testing.T, clientset *fake.Clientset) {
	t.Helper()
	node, err := clientset.CoreV1().Nodes().Get(context.Background(), drainNode, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get node: %v", err)
	}
	if !node.Spec.Unschedulable {
		t.Error("node is not cordoned")
	}
}

func TestDrainNodeEvictsPods(t *testing.T) {
	mirror := drainPod("kube-apiserver", drainNode, "")
	mirror.Annotations = map[string]string{corev1.MirrorPodAnnotationKey: "hash"}
	finished := drainPod("job-done", drainNode, "")
	finished.Status.Phase = corev1.PodSucceeded
	c, clientset := drainClient(t, WritePolicy{}, nil,
		drainPod("web", drainNode, "ReplicaSet"),
		drainPod("db", drainNode, "StatefulSet"),
		drainPod("agent", drainNode, "DaemonSet"),
		mirror,
		finished,
		drainPod("elsewhere", "worker-2", "ReplicaSet"),
	)

	result, err := c.DrainNode(context.Background(), NodeOperationOptions{NodeName: drainNode, IgnoreDaemonSets: true})
	if err != nil {
		t.Fatalf("DrainNode() error = %v", err)
	}
	assertCordoned(t, clientset)
	slices.Sort(result.Evicted)
	if want := []string{"apps/db", "apps/job-done", webPod}; !slices.Equal(result.Evicted, want) {
		t.Errorf("Evicted = %v, want %v", result.Evicted, want)
	}
	slices.Sort(result.Skipped)
	if want := []string{"apps/agent", "apps/kube-apiserver"}; !slices.Equal(result.Skipped, want) {
		t.Errorf("Skipped = %v, want %v", result.Skipped, want)
	}
	for _, name := range []string{"web", "db", "job-done"} {
		if podExists(t, clientset, name) {
			t.Errorf("pod %s is still there after the drain", name)
		}
	}
	for _, name := range []string{"agent", "kube-apiserver", "elsewhere"} {
		if !podExists(t, clientset, name) {
			t.Errorf("pod %s was removed; the drain must leave it", name)
		}
	}
}

// TestDrainNodeRefusesLosses pins the kubectl drain safeguards: a pod that
// would be lost without the option accepting the loss stops the drain before
// any eviction.
func TestDrainNodeRefusesLosses(t *testing.T) {
	withData := drainPod("cache", drainNode, "ReplicaSet")
	withData.Spec.Volumes = []corev1.Volume{{Name: "scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
	for name, tc := range map[string]struct {
		pod  *corev1.Pod
		opts NodeOperationOptions
		want string
	}{
		"DaemonSet pod":   {drainPod("agent", drainNode, "DaemonSet"), NodeOperationOptions{}, "ignore_daemonsets"},
		"unmanaged pod":   {drainPod("bare", drainNode, ""), NodeOperationOptions{IgnoreDaemonSets: true}, "force"},
		"emptyDir volume": {withData, NodeOperationOptions{IgnoreDaemonSets: true, Force: true}, "delete_local_data"},
	} {
		t.Run(name, func(t *testing.T) {
			c, clientset := drainClient(t, WritePolicy{}, nil, drainPod("web", drainNode, "ReplicaSet"), tc.pod)
			tc.opts.NodeName = drainNode
			_, err := c.DrainNode(context.Background(), tc.opts)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("DrainNode() error = %v, want a refusal naming %s", err, tc.want)
			}
			assertCordoned(t, clientset)
			if got := evictions(clientset); len(got) != 0 {
				t.Errorf("evicted %v before refusing", got)
			}
		})
	}

	t.Run("options accept the losses", func(t *testing.T) {
		c, clientset := drainClient(t, WritePolicy{}, nil, drainPod("bare", drainNode, ""), withData)
		_, err := c.DrainNode(context.Background(), NodeOperationOptions{NodeName: drainNode, Force: true, DeleteLocalData: true})
		if err != nil {
			t.Fatalf("DrainNode() error = %v", err)
		}
		if podExists(t, clientset, "bare") || podExists(t, clientset, "cache") {
			t.Error("forced drain left pods on the node")
		}
	})
}

// TestDrainNodeRespectsPodDisruptionBudgets: an eviction a budget refuses is
// retried until the budget allows it, and never forced past it.
func TestDrainNodeRespectsPodDisruptionBudgets(t *testing.T) {
	t.Run("budget allows later", func(t *testing.T) {
		c, clientset := drainClient(t, WritePolicy{}, map[string]int{"web": 2}, drainPod("web", drainNode, "ReplicaSet"))
		result, err := c.DrainNode(context.Background(), NodeOperationOptions{NodeName: drainNode})
		if err != nil {
			t.Fatalf("DrainNode() error = %v", err)
		}
		if got := evictions(clientset); len(got) != 3 {
			t.Errorf("eviction attempts = %v, want 3 (two refused, one allowed)", got)
		}
		if !slices.Equal(result.Evicted, []string{webPod}) || podExists(t, clientset, "web") {
			t.Errorf("web was not evicted: %+v", result)
		}
	})

	t.Run("budget never allows", func(t *testing.T) {
		c, clientset := drainClient(t, WritePolicy{}, map[string]int{"db": -1},
			drainPod("web", drainNode, "ReplicaSet"), drainPod("db", drainNode, "StatefulSet"))
		result, err := c.DrainNode(context.Background(), NodeOperationOptions{NodeName: drainNode, Timeout: 50 * time.Millisecond})
		if err == nil || !strings.Contains(err.Error(), "PodDisruptionBudget") || !strings.Contains(err.Error(), "apps/db") {
			t.Fatalf("DrainNode() error = %v, want a timeout naming the budget and apps/db", err)
		}
		assertCordoned(t, clientset)
		if !podExists(t, clientset, "db") {
			t.Error("db was removed although its budget refused the eviction")
		}
		if !slices.Equal(result.Evicted, []string{webPod}) {
			t.Errorf("Evicted = %v, want [apps/web]", result.Evicted)
		}
	})
}

func TestDrainNodeReadOnly(t *testing.T) {
	c, clientset := drainClient(t, WritePolicy{ReadOnly: true}, nil, drainPod("web", drainNode, "ReplicaSet"))
	result, err := c.DrainNode(context.Background(), NodeOperationOptions{NodeName: drainNode})
	if !errors.Is(err, ErrReadOnly) || result != nil {
		t.Fatalf("DrainNode() = %+v, %v, want nil, ErrReadOnly", result, err)
	}
	if got := evictions(clientset); len(got) != 0 || !podExists(t, clientset, "web") {
		t.Errorf("read-only drain evicted %v", got)
	}
}
