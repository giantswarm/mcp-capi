package integration_test

import "testing"

func TestCapiAzureGetCluster(t *testing.T) {
	runProviderGetClusterTests(t, providerGetClusterConfig{
		provider:    "azure",
		toolName:    "capi_azure_get_cluster",
		managedKind: "AzureManagedCluster",
	})
}
