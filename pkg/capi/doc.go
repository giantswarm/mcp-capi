// Package capi provides a Go client for interacting with Cluster API (CAPI) resources.
//
// This package implements a comprehensive client for managing Kubernetes clusters
// through the Cluster API, supporting operations on clusters, machines, machine
// deployments, and nodes across multiple infrastructure providers.
//
// # Key Components
//
// Client: The main client struct that provides methods for all CAPI operations.
// It wraps both the standard Kubernetes client and the controller-runtime client
// for optimal interaction with CAPI resources.
//
// Providers: Support for multiple infrastructure providers including AWS, Azure,
// GCP, and vSphere. The client can automatically detect the provider type from
// cluster resources.
//
// # Basic Usage
//
//	// Create a new CAPI client
//	client, err := capi.NewClient("")  // Uses default kubeconfig
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// List all clusters
//	clusters, err := client.ListClusters(ctx, "default")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Get cluster status
//	status, err := client.GetClusterStatus(ctx, "default", "my-cluster")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Cluster Operations
//
// The package provides comprehensive cluster management capabilities:
//   - Create, update, and delete clusters
//   - Scale control plane and worker nodes
//   - Upgrade Kubernetes versions
//   - Pause and resume cluster reconciliation
//   - Move clusters between management clusters
//   - Backup cluster configurations
//
// # Machine Operations
//
// Machine-level operations include:
//   - List and get machine details
//   - Delete machines with proper draining
//   - Trigger machine remediation
//   - Create and manage machine deployments
//   - Scale machine deployments
//   - Perform rolling updates
//
// # Node Operations
//
// The package also provides limited node operations:
//   - Cordon and uncordon nodes
//   - Drain nodes (partial implementation)
//   - Get node status and information
//
// Note: Full node operations require access to the workload cluster,
// which may not always be available from the management cluster context.
//
// # Error Handling
//
// All methods return detailed errors that can be inspected for specific
// failure conditions. The package uses fmt.Errorf with %w for error wrapping,
// allowing errors to be unwrapped and inspected.
//
// # Thread Safety
//
// The Client struct and its methods are thread-safe and can be used
// concurrently from multiple goroutines.
package capi