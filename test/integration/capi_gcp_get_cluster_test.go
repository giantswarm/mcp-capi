package integration_test

import "testing"

func TestCapiGCPGetCluster(t *testing.T) {
	runProviderGetClusterTests(t, providerGetClusterConfig{
		provider:    "gcp",
		toolName:    "capi_gcp_get_cluster",
		managedKind: "GCPManagedCluster",
	})
}
