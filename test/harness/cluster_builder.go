package harness

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1 "sigs.k8s.io/cluster-api/api/core/v1beta1" //nolint:staticcheck // CAPI v1beta1 required until v1beta2 migration
)

// controlPlaneConfig holds the configuration for a control plane resource.
type controlPlaneConfig struct {
	kind     string // e.g., "KubeadmControlPlane"
	version  string
	replicas int32
}

// ClusterBuilder provides a fluent API for building cluster resources with custom properties.
// Similar to ToolCall, it accumulates configuration and queues the operation when finalized.
// It embeds clusterBuilderOp (which itself embeds clusterCreateOptions), so all fields
// are promoted and accessible directly on the builder.
type ClusterBuilder struct {
	harness *Harness
	clusterBuilderOp
}

// customRef holds a custom object reference for InfrastructureRef or ControlPlaneRef.
type customRef struct {
	kind string
	name string
}

// Cluster starts a new cluster builder.
func (h *Harness) Cluster(namespace, name string) *ClusterBuilder {
	return &ClusterBuilder{
		harness: h,
		clusterBuilderOp: clusterBuilderOp{
			clusterCreateOptions: clusterCreateOptions{
				namespace: namespace,
				name:      name,
			},
		},
	}
}

// WithProvider sets the infrastructure provider (aws, azure, gcp, vsphere, vcd).
func (cb *ClusterBuilder) WithProvider(provider string) *ClusterBuilder {
	cb.provider = provider
	return cb
}

// WithPaused sets spec.paused on the cluster.
func (cb *ClusterBuilder) WithPaused(paused bool) *ClusterBuilder {
	cb.paused = paused
	return cb
}

// WithLabels sets metadata.labels on the cluster. These labels are merged with
// the default ClusterNameLabel that is always applied.
func (cb *ClusterBuilder) WithLabels(labels map[string]string) *ClusterBuilder {
	cb.labels = labels
	return cb
}

// WithAnnotations sets metadata.annotations on the cluster.
func (cb *ClusterBuilder) WithAnnotations(annotations map[string]string) *ClusterBuilder {
	cb.annotations = annotations
	return cb
}

// WithPhase sets the cluster phase to apply after creation.
func (cb *ClusterBuilder) WithPhase(phase string) *ClusterBuilder {
	cb.phase = phase
	cb.hasStatus = true
	return cb
}

// WithVersion sets the kubernetes version to apply after creation.
func (cb *ClusterBuilder) WithVersion(version string) *ClusterBuilder {
	cb.version = version
	return cb
}

// WithMachines sets the number of machines to create and how many should be ready.
// Ready machines will have a NodeRef set in their status.
func (cb *ClusterBuilder) WithMachines(total, ready int) *ClusterBuilder {
	if ready > total {
		cb.harness.t.Fatalf("WithMachines: ready (%d) cannot exceed total (%d)", ready, total)
	}
	cb.totalMachines = total
	cb.readyMachines = ready
	return cb
}

// WithCustomInfraRef sets a custom InfrastructureRef with an arbitrary Kind.
// This overrides any provider set via WithProvider.
func (cb *ClusterBuilder) WithCustomInfraRef(kind, name string) *ClusterBuilder {
	cb.customInfraRef = &customRef{kind: kind, name: name}
	return cb
}

// WithControlPlaneRef sets a custom ControlPlaneRef with an arbitrary Kind and name.
// Use this to test non-KubeadmControlPlane control plane types or references
// to control plane resources that don't exist.
func (cb *ClusterBuilder) WithControlPlaneRef(kind, name string) *ClusterBuilder {
	cb.customCPRef = &customRef{kind: kind, name: name}
	return cb
}

// WithControlPlaneReady explicitly sets the ControlPlaneReady status field on the cluster.
func (cb *ClusterBuilder) WithControlPlaneReady(ready bool) *ClusterBuilder {
	cb.controlPlaneReady = &ready
	cb.hasStatus = true
	return cb
}

// WithInfraReady explicitly sets the InfrastructureReady status field on the cluster.
func (cb *ClusterBuilder) WithInfraReady(ready bool) *ClusterBuilder {
	cb.infraReady = &ready
	cb.hasStatus = true
	return cb
}

// ConditionBuilder provides a fluent API for configuring a cluster condition.
type ConditionBuilder struct {
	clusterBuilder *ClusterBuilder
	condType       string
	status         corev1.ConditionStatus
	severity       clusterv1.ConditionSeverity
	reason         string
	message        string
}

// WithCondition starts configuring a condition for this cluster.
func (cb *ClusterBuilder) WithCondition(condType string) *ConditionBuilder {
	return &ConditionBuilder{
		clusterBuilder: cb,
		condType:       condType,
	}
}

// True sets the condition status to True.
func (cb *ConditionBuilder) True() *ConditionBuilder {
	cb.status = corev1.ConditionTrue
	return cb
}

// False sets the condition status to False.
func (cb *ConditionBuilder) False() *ConditionBuilder {
	cb.status = corev1.ConditionFalse
	return cb
}

// Unknown sets the condition status to Unknown.
func (cb *ConditionBuilder) Unknown() *ConditionBuilder {
	cb.status = corev1.ConditionUnknown
	return cb
}

// Severity sets the severity for this condition (Error, Warning, Info).
func (cb *ConditionBuilder) Severity(severity clusterv1.ConditionSeverity) *ConditionBuilder {
	cb.severity = severity
	return cb
}

// Reason sets the reason for this condition.
func (cb *ConditionBuilder) Reason(reason string) *ConditionBuilder {
	cb.reason = reason
	return cb
}

// Message sets the message for this condition.
func (cb *ConditionBuilder) Message(message string) *ConditionBuilder {
	cb.message = message
	return cb
}

// Done returns to the ClusterBuilder to continue configuration.
func (cb *ConditionBuilder) Done() *ClusterBuilder {
	cb.clusterBuilder.conditions = append(cb.clusterBuilder.conditions, clusterv1.Condition{
		Type:               clusterv1.ConditionType(cb.condType),
		Status:             cb.status,
		Severity:           cb.severity,
		Reason:             cb.reason,
		Message:            cb.message,
		LastTransitionTime: metav1.Now(),
	})
	return cb.clusterBuilder
}

// ControlPlaneBuilder provides a fluent API for configuring the control plane.
type ControlPlaneBuilder struct {
	clusterBuilder *ClusterBuilder
	kind           string // e.g., "KubeadmControlPlane"
	version        string
	replicas       int32
}

// WithKubeadmControlPlane starts configuring a KubeadmControlPlane for this cluster.
func (cb *ClusterBuilder) WithKubeadmControlPlane() *ControlPlaneBuilder {
	return &ControlPlaneBuilder{
		clusterBuilder: cb,
		kind:           "KubeadmControlPlane",
		replicas:       1, // sensible default
	}
}

// Version sets the Kubernetes version on the control plane.
func (cpb *ControlPlaneBuilder) Version(version string) *ControlPlaneBuilder {
	cpb.version = version
	return cpb
}

// Replicas sets the number of control plane replicas.
func (cpb *ControlPlaneBuilder) Replicas(replicas int32) *ControlPlaneBuilder {
	cpb.replicas = replicas
	return cpb
}

// Done returns to the ClusterBuilder to continue configuration.
func (cpb *ControlPlaneBuilder) Done() *ClusterBuilder {
	cpb.clusterBuilder.controlPlane = &controlPlaneConfig{
		kind:     cpb.kind,
		version:  cpb.version,
		replicas: cpb.replicas,
	}
	return cpb.clusterBuilder
}

// Create queues the cluster creation operation and returns to the harness.
func (cb *ClusterBuilder) Create() *Harness {
	cb.harness.t.Helper()
	if cb.controlPlane != nil && cb.customCPRef != nil {
		cb.harness.t.Fatalf("ClusterBuilder for %s/%s: cannot set both WithKubeadmControlPlane and WithControlPlaneRef", cb.namespace, cb.name)
	}
	op := cb.clusterBuilderOp // value copy
	cb.harness.operations = append(cb.harness.operations, &op)
	return cb.harness
}
