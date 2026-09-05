package harness

import (
	"os"
	"path/filepath"
	"testing"

	controlplanev1 "sigs.k8s.io/cluster-api/api/controlplane/kubeadm/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1"                      //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
	"sigs.k8s.io/yaml"
)

// TestCoreCRDsStoreTheServedAPIVersion guards the storage version of the CAPI
// core and control-plane CRDs in crds/.
//
// The harness and the product both talk to the API server through the v1beta1
// types, and the test API server runs without the CAPI conversion webhook. With
// `conversion.strategy: None` the API server only rewrites apiVersion and prunes
// every field the storage version's schema does not know. Upstream ships v1beta2
// as the storage version, which silently drops status.controlPlaneReady,
// status.infrastructureReady and the condition severity on every round-trip, so
// no test could ever observe a healthy cluster. The CRDs in crds/ therefore store
// v1beta1; a re-download from upstream must keep that flip.
func TestCoreCRDsStoreTheServedAPIVersion(t *testing.T) {
	t.Parallel()

	crdDir, err := getCRDPath()
	if err != nil {
		t.Fatalf("locating CRD directory: %v", err)
	}

	wantStorage := map[string]string{
		clusterv1.GroupVersion.Group:      clusterv1.GroupVersion.Version,
		controlplanev1.GroupVersion.Group: controlplanev1.GroupVersion.Version,
	}

	files, err := filepath.Glob(filepath.Join(crdDir, "*.yaml"))
	if err != nil {
		t.Fatalf("listing CRDs: %v", err)
	}

	checked := 0
	for _, path := range files {
		raw, err := os.ReadFile(path) //#nosec G304 -- test reads repository fixtures
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		var crd struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Spec struct {
				Group    string `json:"group"`
				Versions []struct {
					Name    string `json:"name"`
					Served  bool   `json:"served"`
					Storage bool   `json:"storage"`
				} `json:"versions"`
			} `json:"spec"`
		}
		if err := yaml.Unmarshal(raw, &crd); err != nil {
			t.Fatalf("parsing %s: %v", path, err)
		}

		want, ok := wantStorage[crd.Spec.Group]
		if !ok {
			continue
		}
		checked++

		storage := ""
		served := false
		for _, v := range crd.Spec.Versions {
			if v.Storage {
				storage = v.Name
			}
			if v.Name == want && v.Served {
				served = true
			}
		}
		if storage != want {
			t.Errorf("%s: storage version is %q, want %q: the test API server has no conversion webhook, so every %s-only field is pruned on write", crd.Metadata.Name, storage, want, want)
		}
		if !served {
			t.Errorf("%s: version %q must be served", crd.Metadata.Name, want)
		}
	}

	if checked == 0 {
		t.Fatalf("no CAPI core or control-plane CRDs found in %s", crdDir)
	}
}
