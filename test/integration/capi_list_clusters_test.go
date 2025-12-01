package integration_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"

	"github.com/giantswarm/mcp-capi/test/harness"
)

var providers = []string{"aws", "azure", "gcp", "vsphere", "vcd", "nonexistent"}

var _ = Describe("capi_list_clusters", func() {
	It("lists multiple clusters", func() {
		namespace := "test-clusters"

		harness.New(GinkgoT()).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster-1", "test-cluster-2", "test-cluster-3").
			ToolCall("capi_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("multiple.golden").
			Execute()
	})

	It("lists clusters with metadata", func() {
		namespace := "test-clusters"

		harness.New(GinkgoT()).
			CreateNamespace(namespace).
			CreateClusters(namespace, "test-cluster-1", "test-cluster-2", "test-cluster-3").
			ToolCall("capi_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("metadata.golden").
			Execute()
	})

	It("filters clusters by namespace", func() {
		namespace := "test-clusters"
		otherNamespace := "other-clusters"

		harness.New(GinkgoT()).
			CreateNamespace(namespace).
			CreateClusters(namespace, "cluster-in-test-ns").
			CreateNamespace(otherNamespace).
			CreateClusters(otherNamespace, "cluster-in-other-ns").
			ToolCall("capi_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("namespace_filter_test.golden").
			ToolCall("capi_list_clusters").
			WithArg("namespace", otherNamespace).
			AssertContent("namespace_filter_other.golden").
			Execute()
	})

	It("handles empty namespace", func() {
		namespace := "test-clusters"

		harness.New(GinkgoT()).
			CreateNamespace(namespace).
			ToolCall("capi_list_clusters").
			WithArg("namespace", namespace).
			AssertContent("empty.golden").
			Execute()
	})

	It("lists clusters across all namespaces", func() {
		namespace := "test-clusters"
		namespace1 := "multi-ns-1"
		namespace2 := "multi-ns-2"

		harness.New(GinkgoT()).
			CreateNamespace(namespace).
			CreateClusters(namespace, "cluster-ns1").
			CreateNamespace(namespace1).
			CreateClusters(namespace1, "cluster-ns2").
			CreateNamespace(namespace2).
			CreateClusters(namespace2, "cluster-ns3").
			ToolCall("capi_list_clusters").
			WithArg("namespace", "").
			AssertContent("all_namespaces.golden").
			Execute()
	})

	for _, provider := range providers {
		provider := provider // capture loop variable

		It(fmt.Sprintf("lists %s clusters from same namespace", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-cluster-1").WithProvider(provider).Create().
				Cluster(namespace, provider+"-cluster-2").WithProvider(provider).Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_same_namespace.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s clusters from different namespaces", provider), func() {
			ns1 := provider + "-ns-1"
			ns2 := provider + "-ns-2"
			harness.New(GinkgoT()).
				CreateNamespace(ns1).
				Cluster(ns1, provider+"-cluster-1").WithProvider(provider).Create().
				CreateNamespace(ns2).
				Cluster(ns2, provider+"-cluster-2").WithProvider(provider).Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", "").
				AssertContent(provider + "_different_namespaces.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s clusters with different phases", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-pending").WithProvider(provider).WithPhase("Pending").Create().
				Cluster(namespace, provider+"-provisioning").WithProvider(provider).WithPhase("Provisioning").Create().
				Cluster(namespace, provider+"-provisioned").WithProvider(provider).WithPhase("Provisioned").Create().
				Cluster(namespace, provider+"-deleting").WithProvider(provider).WithPhase("Deleting").Create().
				Cluster(namespace, provider+"-failed").WithProvider(provider).WithPhase("Failed").Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_different_phases.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s clusters with different versions", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-v128").WithProvider(provider).WithVersion("v1.28.0").Create().
				Cluster(namespace, provider+"-v129").WithProvider(provider).WithVersion("v1.29.0").Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_different_versions.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s clusters with partial machine readiness", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-partial").WithProvider(provider).WithMachines(2, 1).Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_partial_machines.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s clusters with all machines ready", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-all-ready").WithProvider(provider).WithMachines(3, 3).Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_all_machines_ready.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s clusters with no machines ready", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-none-ready").WithProvider(provider).WithMachines(5, 0).Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_no_machines_ready.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s clusters with version from control plane", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-cluster-1").
				WithProvider(provider).
				WithKubeadmControlPlane().Version("v1.28.0").Done().
				Create().
				Cluster(namespace, provider+"-cluster-2").
				WithProvider(provider).
				WithKubeadmControlPlane().Version("v1.31.0").Done().
				Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_control_plane_version.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s cluster with version precedence over control plane", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-cluster").
				WithProvider(provider).
				WithVersion("v1.28.0").
				WithKubeadmControlPlane().Version("v1.99.0").Done().
				Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_version_precedence.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s clusters with different conditions", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				// Cluster 1: All conditions True (healthy cluster)
				Cluster(namespace, provider+"-healthy").WithProvider(provider).
				WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is fully operational").Done().
				WithCondition("ControlPlaneReady").True().Reason("ControlPlaneInitialized").Message("Control plane is ready").Done().
				WithCondition("InfrastructureReady").True().Reason("InfrastructureProvisioned").Message("Infrastructure is ready").Done().
				Create().
				// Cluster 2: Mixed conditions (provisioning cluster)
				Cluster(namespace, provider+"-provisioning").WithProvider(provider).
				WithCondition("Ready").False().Reason("WaitingForControlPlane").Message("Waiting for control plane to be ready").Done().
				WithCondition("ControlPlaneReady").False().Reason("ScalingUp").Message("Control plane is scaling up").Done().
				WithCondition("InfrastructureReady").True().Reason("InfrastructureProvisioned").Message("Infrastructure is ready").Done().
				Create().
				// Cluster 3: Unhealthy cluster
				Cluster(namespace, provider+"-unhealthy").WithProvider(provider).
				WithCondition("Ready").False().Reason("ClusterNotReady").Message("Cluster has issues").Done().
				WithCondition("ControlPlaneReady").False().Reason("ControlPlaneUnhealthy").Message("Control plane has unhealthy replicas").Done().
				WithCondition("InfrastructureReady").False().Reason("WaitingForInfrastructure").Message("Infrastructure provisioning failed").Done().
				Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_different_conditions.golden").
				Execute()
		})

		It(fmt.Sprintf("lists %s clusters with all properties", provider), func() {
			namespace := "test-clusters"
			harness.New(GinkgoT()).
				CreateNamespace(namespace).
				Cluster(namespace, provider+"-cluster-1").WithProvider(provider).WithPhase("Provisioned").WithVersion("v1.28.0").WithMachines(3, 3).
					WithCondition("Ready").True().Reason("ClusterReady").Message("Cluster is fully operational").Done().
					WithCondition("ControlPlaneReady").True().Reason("ControlPlaneInitialized").Message("Control plane is ready").Done().
					WithCondition("InfrastructureReady").True().Reason("InfrastructureProvisioned").Message("Infrastructure is ready").Done().
					Create().
				Cluster(namespace, provider+"-cluster-2").WithProvider(provider).WithPhase("Provisioning").WithVersion("v1.29.0").WithMachines(2, 1).
					WithCondition("Ready").False().Reason("WaitingForControlPlane").Message("Waiting for control plane to be ready").Done().
					WithCondition("ControlPlaneReady").False().Reason("ScalingUp").Message("Control plane is scaling up").Done().
					WithCondition("InfrastructureReady").True().Reason("InfrastructureProvisioned").Message("Infrastructure is ready").Done().
					Create().
				Cluster(namespace, provider+"-cluster-3").WithProvider(provider).WithPhase("Pending").WithVersion("v1.30.0").WithMachines(5, 0).
					WithCondition("Ready").False().Reason("ClusterNotReady").Message("Cluster is not ready").Done().
					WithCondition("ControlPlaneReady").False().Reason("WaitingForInfrastructure").Message("Waiting for infrastructure").Done().
					WithCondition("InfrastructureReady").False().Reason("Provisioning").Message("Infrastructure is being provisioned").Done().
					Create().
				ToolCall("capi_list_clusters").
				WithArg("namespace", namespace).
				AssertContent(provider + "_multiple_all_properties.golden").
				Execute()
		})

	}

	It("lists clusters with all providers across multiple namespaces", func() {
		namespace1 := "provider-ns-1"
		namespace2 := "provider-ns-2"
		namespace3 := "provider-ns-3"

		harness.New(GinkgoT()).
			// Namespace 1: 5 clusters (AWS x2, Azure x2, GCP x1)
			CreateNamespace(namespace1).
			Cluster(namespace1, "aws-cluster-1").WithProvider("aws").WithPhase("Provisioned").WithVersion("v1.28.0").WithMachines(3, 3).Create().
			Cluster(namespace1, "aws-cluster-2").WithProvider("aws").WithPhase("Provisioning").WithVersion("v1.29.0").WithMachines(4, 2).Create().
			Cluster(namespace1, "azure-cluster-1").WithProvider("azure").WithPhase("Provisioned").WithVersion("v1.28.0").WithMachines(5, 5).Create().
			Cluster(namespace1, "azure-cluster-2").WithProvider("azure").WithPhase("Pending").WithVersion("v1.30.0").WithMachines(3, 0).Create().
			Cluster(namespace1, "gcp-cluster-1").WithProvider("gcp").WithPhase("Provisioned").WithVersion("v1.29.0").WithMachines(4, 4).Create().
			// Namespace 2: 5 clusters (GCP x1, vSphere x2, VCD x2)
			CreateNamespace(namespace2).
			Cluster(namespace2, "gcp-cluster-2").WithProvider("gcp").WithPhase("Deleting").WithVersion("v1.28.0").WithMachines(2, 2).Create().
			Cluster(namespace2, "vsphere-cluster-1").WithProvider("vsphere").WithPhase("Provisioned").WithVersion("v1.29.0").WithMachines(6, 6).Create().
			Cluster(namespace2, "vsphere-cluster-2").WithProvider("vsphere").WithPhase("Failed").WithVersion("v1.28.0").WithMachines(4, 0).Create().
			Cluster(namespace2, "vcd-cluster-1").WithProvider("vcd").WithPhase("Provisioned").WithVersion("v1.30.0").WithMachines(3, 3).Create().
			Cluster(namespace2, "vcd-cluster-2").WithProvider("vcd").WithPhase("Provisioning").WithVersion("v1.29.0").WithMachines(5, 1).Create().
			// Namespace 3: 5 clusters (AWS x1, Azure x1, GCP x1, vSphere x1, VCD x1)
			CreateNamespace(namespace3).
			Cluster(namespace3, "aws-cluster-3").WithProvider("aws").WithPhase("Failed").WithVersion("v1.27.0").WithMachines(2, 0).Create().
			Cluster(namespace3, "azure-cluster-3").WithProvider("azure").WithPhase("Provisioned").WithVersion("v1.29.0").WithMachines(4, 4).Create().
			Cluster(namespace3, "gcp-cluster-3").WithProvider("gcp").WithPhase("Pending").WithVersion("v1.30.0").WithMachines(6, 0).Create().
			Cluster(namespace3, "vsphere-cluster-3").WithProvider("vsphere").WithPhase("Provisioning").WithVersion("v1.28.0").WithMachines(5, 3).Create().
			Cluster(namespace3, "vcd-cluster-3").WithProvider("vcd").WithPhase("Deleting").WithVersion("v1.29.0").WithMachines(2, 2).Create().
			// List all clusters across all namespaces
			ToolCall("capi_list_clusters").
			WithArg("namespace", "").
			AssertContent("all_providers_multiple_namespaces.golden").
			Execute()
	})
})
