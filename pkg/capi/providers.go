package capi

import (
	"context"
	"fmt"

	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Provider represents an infrastructure provider
type Provider string

const (
	ProviderAWS     Provider = "aws"
	ProviderAzure   Provider = "azure"
	ProviderGCP     Provider = "gcp"
	ProviderVSphere Provider = "vsphere"
	ProviderUnknown Provider = "unknown"
)

// InitializeProviders adds all provider schemes to the client
func (c *Client) InitializeProviders() error {
	scheme := c.ctrlClient.Scheme()

	// Add KubeadmControlPlane scheme
	if err := controlplanev1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("failed to add KubeadmControlPlane to scheme: %w", err)
	}

	// Note: Infrastructure provider schemes would be added here
	// For now, we'll use unstructured resources for provider-specific resources

	return nil
}

// GetProviderForCluster determines which infrastructure provider a cluster is using
func (c *Client) GetProviderForCluster(ctx context.Context, namespace, clusterName string) (Provider, error) {
	cluster, err := c.GetCluster(ctx, namespace, clusterName)
	if err != nil {
		return ProviderUnknown, err
	}

	if cluster.Spec.InfrastructureRef == nil {
		return ProviderUnknown, fmt.Errorf("cluster has no infrastructure reference")
	}

	// Determine provider based on the infrastructure reference kind
	switch cluster.Spec.InfrastructureRef.Kind {
	case "AWSCluster":
		return ProviderAWS, nil
	case "AzureCluster":
		return ProviderAzure, nil
	case "GCPCluster":
		return ProviderGCP, nil
	case "VSphereCluster":
		return ProviderVSphere, nil
	default:
		return ProviderUnknown, nil
	}
}

// GetKubeadmControlPlane retrieves the KubeadmControlPlane for a cluster
func (c *Client) GetKubeadmControlPlane(ctx context.Context, namespace, name string) (*controlplanev1.KubeadmControlPlane, error) {
	kcp := &controlplanev1.KubeadmControlPlane{}
	key := client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}

	if err := c.ctrlClient.Get(ctx, key, kcp); err != nil {
		return nil, fmt.Errorf("failed to get KubeadmControlPlane %s/%s: %w", namespace, name, err)
	}

	return kcp, nil
}

// ListKubeadmControlPlanes lists all KubeadmControlPlanes
func (c *Client) ListKubeadmControlPlanes(ctx context.Context, namespace string) (*controlplanev1.KubeadmControlPlaneList, error) {
	kcpList := &controlplanev1.KubeadmControlPlaneList{}

	opts := []client.ListOption{}
	if namespace != "" {
		opts = append(opts, client.InNamespace(namespace))
	}

	if err := c.ctrlClient.List(ctx, kcpList, opts...); err != nil {
		return nil, fmt.Errorf("failed to list KubeadmControlPlanes: %w", err)
	}

	return kcpList, nil
}

// GetInfrastructureResource retrieves an infrastructure-specific resource as unstructured
func (c *Client) GetInfrastructureResource(ctx context.Context, ref *client.ObjectKey, into client.Object) error {
	if err := c.ctrlClient.Get(ctx, *ref, into); err != nil {
		return fmt.Errorf("failed to get infrastructure resource: %w", err)
	}
	return nil
}

// ScaleControlPlane scales a KubeadmControlPlane to the specified number of replicas
func (c *Client) ScaleControlPlane(ctx context.Context, namespace, name string, replicas int32) error {
	kcp, err := c.GetKubeadmControlPlane(ctx, namespace, name)
	if err != nil {
		return err
	}

	// Update replicas
	kcp.Spec.Replicas = &replicas

	if err := c.ctrlClient.Update(ctx, kcp); err != nil {
		return fmt.Errorf("failed to scale control plane: %w", err)
	}

	return nil
}

// ScaleCluster scales either control plane or worker nodes of a cluster
func (c *Client) ScaleCluster(ctx context.Context, namespace, clusterName, target string, replicas int, machineDeploymentName string) error {
	switch target {
	case "controlplane":
		return c.ScaleControlPlane(ctx, namespace, clusterName, int32(replicas))
	case "workers":
		if machineDeploymentName == "" {
			return fmt.Errorf("machineDeployment name is required when scaling workers")
		}
		return c.ScaleMachineDeployment(ctx, namespace, machineDeploymentName, int32(replicas))
	default:
		return fmt.Errorf("invalid target: %s (must be 'controlplane' or 'workers')", target)
	}
}

// ScaleMachineDeployment scales a MachineDeployment to the specified number of replicas
func (c *Client) ScaleMachineDeployment(ctx context.Context, namespace, name string, replicas int32) error {
	md, err := c.GetMachineDeployment(ctx, namespace, name)
	if err != nil {
		return err
	}

	// Update replicas
	md.Spec.Replicas = &replicas

	if err := c.ctrlClient.Update(ctx, md); err != nil {
		return fmt.Errorf("failed to scale machine deployment: %w", err)
	}

	return nil
}
