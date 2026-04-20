package integration_test

import (
	"fmt"
	"testing"

	"github.com/giantswarm/mcp-capi/test/harness"
)

// runProviderClusterDetailsTests runs the per-provider "all properties" and
// "control plane version" subtests for the given toolName.
func runProviderClusterDetailsTests(t *testing.T, toolName, verbPhrase string) {
	t.Helper()

	for _, provider := range providers {
		t.Run(fmt.Sprintf("%s %s cluster with all properties", verbPhrase, provider), func(t *testing.T) {
			t.Parallel()
			namespace := testNamespace

			harness.New(t).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-full-cluster").
				WithProvider(provider).WithPhase("Provisioned").WithVersion("v1.29.0").WithMachines(3, 2).
				WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is fully operational").Done().
				WithCondition("ControlPlaneReady").
				True().Reason("ControlPlaneInitialized").Message("Control plane is ready").Done().
				WithCondition("InfrastructureReady").
				True().Reason("InfrastructureProvisioned").Message("Infrastructure is ready").Done().
				Create().
				ToolCall(toolName).
				WithArg("namespace", namespace).
				WithArg("name", provider+"-full-cluster").
				AssertContent(provider + "_all_properties.golden").
				Execute()
		})

		t.Run(fmt.Sprintf("%s %s cluster with version from control plane", verbPhrase, provider), func(t *testing.T) {
			t.Parallel()
			namespace := testNamespace

			harness.New(t).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-kcp-cluster").
				WithProvider(provider).
				WithKubeadmControlPlane().Version("v1.30.0").Done().
				Create().
				ToolCall(toolName).
				WithArg("namespace", namespace).
				WithArg("name", provider+"-kcp-cluster").
				AssertContent(provider + "_control_plane_version.golden").
				Execute()
		})
	}
}
