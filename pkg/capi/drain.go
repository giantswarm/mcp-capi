package capi

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

// DefaultDrainTimeout bounds a drain whose options name no timeout.
const DefaultDrainTimeout = 2 * time.Minute

// drainPollInterval is how often a drain retries the evictions a
// PodDisruptionBudget refuses and checks whether evicted pods are gone.
var drainPollInterval = 5 * time.Second

// DrainResult reports what a drain did to the pods of a node.
type DrainResult struct {
	NodeName string
	// Evicted are the evicted pods, as namespace/name, all gone from the
	// node when DrainNode returns without an error.
	Evicted []string
	// Skipped are the pods left on the node: mirror pods, and DaemonSet
	// pods when IgnoreDaemonSets is set.
	Skipped []string
}

// DrainNode cordons a node, then evicts its pods through the Eviction API so
// that every PodDisruptionBudget is respected: an eviction a budget refuses is
// retried until the budget allows it or opts.Timeout (DefaultDrainTimeout when
// zero) runs out. It returns once every evicted pod is gone from the node.
//
// Like kubectl drain, it evicts nothing when a pod would be lost without the
// option that accepts the loss: DaemonSet pods need IgnoreDaemonSets (they are
// then left on the node), pods with emptyDir volumes need DeleteLocalData and
// pods no controller manages need Force. Mirror pods are always left, and
// finished pods are always evicted. The node stays cordoned whatever the
// outcome; the result reports what was done before an error.
func (c *Client) DrainNode(ctx context.Context, opts NodeOperationOptions) (*DrainResult, error) {
	nodeName, err := c.resolveNodeName(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := c.setUnschedulable(ctx, nodeName, true); err != nil {
		return nil, err
	}
	result := &DrainResult{NodeName: nodeName}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultDrainTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pods, err := c.k8sClient.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
	})
	if err != nil {
		return result, fmt.Errorf("failed to list the pods of node %s: %w", nodeName, err)
	}
	evict, skipped, refusals := drainPlan(pods.Items, nodeName, opts)
	result.Skipped = skipped
	if len(refusals) > 0 {
		return result, fmt.Errorf("refusing to drain node %s, no pod was evicted: %s", nodeName, strings.Join(refusals, "; "))
	}

	evicted, err := c.evictPods(ctx, evict, opts.GracePeriodSeconds)
	for _, pod := range evicted {
		result.Evicted = append(result.Evicted, podName(&pod))
	}
	if err != nil {
		return result, err
	}
	return result, c.waitForPodsGone(ctx, evicted)
}

// drainPlan sorts the pods of nodeName into those to evict and those to
// leave, and names each pod that opts do not allow to evict or leave.
func drainPlan(pods []corev1.Pod, nodeName string, opts NodeOperationOptions) (evict []corev1.Pod, skipped, refusals []string) {
	for _, pod := range pods {
		if pod.Spec.NodeName != nodeName {
			continue
		}
		name := podName(&pod)
		if _, mirror := pod.Annotations[corev1.MirrorPodAnnotationKey]; mirror {
			skipped = append(skipped, name)
			continue
		}
		if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			evict = append(evict, pod)
			continue
		}
		controller := metav1.GetControllerOf(&pod)
		switch {
		case controller != nil && controller.Kind == "DaemonSet":
			if opts.IgnoreDaemonSets {
				skipped = append(skipped, name)
			} else {
				refusals = append(refusals, fmt.Sprintf("pod %s belongs to a DaemonSet (set ignore_daemonsets to leave it)", name))
			}
			continue
		case controller == nil && !opts.Force:
			refusals = append(refusals, fmt.Sprintf("pod %s is managed by no controller and would be lost (set force)", name))
			continue
		case hasLocalStorage(&pod) && !opts.DeleteLocalData:
			refusals = append(refusals, fmt.Sprintf("pod %s has emptyDir data that would be lost (set delete_local_data)", name))
			continue
		}
		evict = append(evict, pod)
	}
	return evict, skipped, refusals
}

// evictPods evicts pods, retrying those a PodDisruptionBudget refuses every
// drainPollInterval until ctx is done. It returns the pods evicted so far.
func (c *Client) evictPods(ctx context.Context, pods []corev1.Pod, gracePeriodSeconds *int64) ([]corev1.Pod, error) {
	var evicted []corev1.Pod
	pending := pods
	for {
		var refused []corev1.Pod
		for _, pod := range pending {
			err := c.k8sClient.PolicyV1().Evictions(pod.Namespace).Evict(ctx, &policyv1.Eviction{
				ObjectMeta:    metav1.ObjectMeta{Namespace: pod.Namespace, Name: pod.Name},
				DeleteOptions: &metav1.DeleteOptions{GracePeriodSeconds: gracePeriodSeconds},
			})
			switch {
			case err == nil, apierrors.IsNotFound(err):
				evicted = append(evicted, pod)
			case apierrors.IsTooManyRequests(err):
				refused = append(refused, pod)
			default:
				return evicted, fmt.Errorf("failed to evict pod %s: %w", podName(&pod), err)
			}
		}
		if len(refused) == 0 {
			return evicted, nil
		}
		select {
		case <-ctx.Done():
			return evicted, fmt.Errorf("the drain timed out while a PodDisruptionBudget still refused to evict %s", podNames(refused))
		case <-time.After(drainPollInterval):
		}
		pending = refused
	}
}

// waitForPodsGone waits until every pod is deleted or replaced by a pod of
// the same name, checking every drainPollInterval until ctx is done.
func (c *Client) waitForPodsGone(ctx context.Context, pods []corev1.Pod) error {
	pending := pods
	for {
		var remaining []corev1.Pod
		for _, pod := range pending {
			current, err := c.k8sClient.CoreV1().Pods(pod.Namespace).Get(ctx, pod.Name, metav1.GetOptions{})
			switch {
			case apierrors.IsNotFound(err):
			case err != nil:
				return fmt.Errorf("failed to check whether pod %s is gone: %w", podName(&pod), err)
			case current.UID == pod.UID:
				remaining = append(remaining, pod)
			}
		}
		if len(remaining) == 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("the drain timed out while evicted pods were still terminating: %s", podNames(remaining))
		case <-time.After(drainPollInterval):
		}
		pending = remaining
	}
}

func hasLocalStorage(pod *corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		if volume.EmptyDir != nil {
			return true
		}
	}
	return false
}

func podName(pod *corev1.Pod) string {
	return pod.Namespace + "/" + pod.Name
}

func podNames(pods []corev1.Pod) string {
	names := make([]string, 0, len(pods))
	for i := range pods {
		names = append(names, podName(&pods[i]))
	}
	return strings.Join(names, ", ")
}
