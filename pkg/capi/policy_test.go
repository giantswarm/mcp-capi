package capi

import (
	"errors"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
)

const (
	testNamespace     = "org-giantswarm"
	testCluster       = "gazelle"
	testKustomization = "clusters"
)

func clusterWith(labels, annotations map[string]string) *clusterv1.Cluster {
	return &clusterv1.Cluster{ObjectMeta: metav1.ObjectMeta{
		Namespace:   testNamespace,
		Name:        testCluster,
		Labels:      labels,
		Annotations: annotations,
	}}
}

func TestManagedBy(t *testing.T) {
	cases := map[string]struct {
		labels      map[string]string
		annotations map[string]string
		wantKind    string
		wantName    string
	}{
		"unmanaged": {},
		"unrelated labels only": {
			labels: map[string]string{"cluster.x-k8s.io/cluster-name": testCluster, managedByLabel: "kustomize"},
		},
		"flux kustomization": {
			labels:   map[string]string{fluxKustomizationNameLabel: testKustomization, fluxKustomizationNamespaceLabel: "flux-giantswarm"},
			wantKind: "Flux Kustomization",
			wantName: "flux-giantswarm/" + testKustomization,
		},
		"flux helmrelease": {
			labels:   map[string]string{fluxHelmReleaseNameLabel: testCluster, fluxHelmReleaseNamespaceLabel: testNamespace},
			wantKind: "Flux HelmRelease",
			wantName: testNamespace + "/" + testCluster,
		},
		"argo label": {
			labels:   map[string]string{argoInstanceLabel: testKustomization},
			wantKind: "Argo CD Application",
			wantName: testKustomization,
		},
		"argo annotation": {
			annotations: map[string]string{argoTrackingAnnotation: "clusters:cluster.x-k8s.io/Cluster:org-giantswarm/gazelle"},
			wantKind:    "Argo CD Application",
			wantName:    "clusters:cluster.x-k8s.io/Cluster:org-giantswarm/gazelle",
		},
		"helm release as rendered by cluster-aws on a management cluster": {
			labels:      map[string]string{managedByLabel: managedByHelm, "helm.sh/chart": "cluster-8.0.0"},
			annotations: map[string]string{helmReleaseNameAnnotation: testCluster, helmReleaseNamespaceAnnotation: testNamespace},
			wantKind:    "Helm release",
			wantName:    "org-giantswarm/gazelle",
		},
		"helm managed-by label only": {
			labels:   map[string]string{managedByLabel: managedByHelm},
			wantKind: "Helm release",
		},
		"flux wins over helm": {
			labels:      map[string]string{fluxKustomizationNameLabel: testKustomization, managedByLabel: managedByHelm},
			annotations: map[string]string{helmReleaseNameAnnotation: testCluster},
			wantKind:    "Flux Kustomization",
			wantName:    "clusters",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			owner, managed := ManagedBy(clusterWith(tc.labels, tc.annotations))
			if managed != (tc.wantKind != "") {
				t.Fatalf("ManagedBy() managed = %v, want %v (owner %+v)", managed, tc.wantKind != "", owner)
			}
			if owner.Kind != tc.wantKind || owner.Name != tc.wantName {
				t.Fatalf("ManagedBy() = %+v, want kind %q name %q", owner, tc.wantKind, tc.wantName)
			}
			if managed && (owner.Marker == "" || owner.Advice == "") {
				t.Fatalf("ManagedBy() = %+v, want marker and advice", owner)
			}
		})
	}
}

func TestWritePolicy(t *testing.T) {
	helmLabels := map[string]string{managedByLabel: managedByHelm}
	helmAnnotations := map[string]string{helmReleaseNameAnnotation: testCluster, helmReleaseNamespaceAnnotation: testNamespace}
	managed := clusterWith(helmLabels, helmAnnotations)
	unmanaged := clusterWith(map[string]string{"cluster.x-k8s.io/cluster-name": testCluster}, nil)
	protected := clusterWith(map[string]string{PreventDeletionLabel: "true"}, nil)
	managedAndProtected := clusterWith(map[string]string{managedByLabel: managedByHelm, PreventDeletionLabel: "true"}, nil)

	permissive := WritePolicy{}
	guarded := WritePolicy{GitOpsGuard: true}
	readOnly := WritePolicy{ReadOnly: true, GitOpsGuard: true}

	t.Run("permissive allows everything except protected deletions", func(t *testing.T) {
		if err := permissive.CheckCreate("Cluster", "org", "new"); err != nil {
			t.Fatalf("CheckCreate() = %v", err)
		}
		if err := permissive.CheckUpdate("Cluster", managed); err != nil {
			t.Fatalf("CheckUpdate(managed) = %v", err)
		}
		if err := permissive.CheckDelete("Cluster", managed); err != nil {
			t.Fatalf("CheckDelete(managed) = %v", err)
		}
		err := permissive.CheckDelete("Cluster", protected)
		if !errors.Is(err, ErrDeletionPrevented) {
			t.Fatalf("CheckDelete(protected) = %v, want ErrDeletionPrevented", err)
		}
		if !strings.Contains(err.Error(), PreventDeletionLabel) {
			t.Fatalf("CheckDelete(protected) = %q, want the label named", err)
		}
	})

	t.Run("gitops guard refuses managed objects and names the owner", func(t *testing.T) {
		if err := guarded.CheckCreate("Cluster", "org", "new"); err != nil {
			t.Fatalf("CheckCreate() = %v, creates are not guarded", err)
		}
		if err := guarded.CheckUpdate("Cluster", unmanaged); err != nil {
			t.Fatalf("CheckUpdate(unmanaged) = %v", err)
		}
		if err := guarded.CheckDelete("Cluster", unmanaged); err != nil {
			t.Fatalf("CheckDelete(unmanaged) = %v", err)
		}
		err := guarded.CheckUpdate("Cluster", managed)
		if !errors.Is(err, ErrManagedResource) {
			t.Fatalf("CheckUpdate(managed) = %v, want ErrManagedResource", err)
		}
		for _, want := range []string{"refusing to update Cluster org-giantswarm/gazelle", "Helm release org-giantswarm/gazelle", helmReleaseNameAnnotation, "in Git"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("CheckUpdate(managed) = %q, want %q in it", err, want)
			}
		}
		if err := guarded.CheckDelete("Cluster", managed); !errors.Is(err, ErrManagedResource) {
			t.Fatalf("CheckDelete(managed) = %v, want ErrManagedResource", err)
		}
		// The GitOps refusal comes first; the protection label is reported when the object is otherwise deletable.
		if err := guarded.CheckDelete("Cluster", managedAndProtected); !errors.Is(err, ErrManagedResource) {
			t.Fatalf("CheckDelete(managed+protected) = %v, want ErrManagedResource", err)
		}
		if err := guarded.CheckDelete("Cluster", protected); !errors.Is(err, ErrDeletionPrevented) {
			t.Fatalf("CheckDelete(protected) = %v, want ErrDeletionPrevented", err)
		}
	})

	t.Run("read-only refuses every mutation", func(t *testing.T) {
		if err := readOnly.CheckCreate("MachineDeployment", "org", "pool"); !errors.Is(err, ErrReadOnly) {
			t.Fatalf("CheckCreate() = %v, want ErrReadOnly", err)
		}
		err := readOnly.CheckUpdate("Cluster", unmanaged)
		if !errors.Is(err, ErrReadOnly) {
			t.Fatalf("CheckUpdate(unmanaged) = %v, want ErrReadOnly", err)
		}
		if !strings.Contains(err.Error(), "--read-only=false") {
			t.Fatalf("CheckUpdate() = %q, want the flag to turn it off named", err)
		}
		if err := readOnly.CheckDelete("Cluster", unmanaged); !errors.Is(err, ErrReadOnly) {
			t.Fatalf("CheckDelete(unmanaged) = %v, want ErrReadOnly", err)
		}
	})
}
