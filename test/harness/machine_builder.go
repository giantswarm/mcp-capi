package harness

import (
	corev1 "k8s.io/api/core/v1"
)

// machineCondition holds a machine condition configuration.
type machineCondition struct {
	condType string
	status   corev1.ConditionStatus
	reason   string
	message  string
}

// machineAddress holds a machine address configuration.
type machineAddress struct {
	addrType string
	address  string
}

// MachineBuilder provides a fluent API for building individual CAPI Machine resources.
// Use this for fine-grained control over machine fields like Bootstrap.ConfigRef,
// InfrastructureRef, Version, and Conditions.
type MachineBuilder struct {
	harness       *Harness
	namespace     string
	name          string
	clusterName   string
	phase         string
	version       string
	providerID    string
	nodeRefName   string
	configRefKind string
	configRefName string
	infraRefKind  string
	infraRefName  string
	conditions    []machineCondition
	addresses     []machineAddress
}

// Machine starts a new machine builder.
func (h *Harness) Machine(namespace, name string) *MachineBuilder {
	return &MachineBuilder{
		harness:   h,
		namespace: namespace,
		name:      name,
	}
}

// ForCluster sets the cluster name for this machine.
func (mb *MachineBuilder) ForCluster(clusterName string) *MachineBuilder {
	mb.clusterName = clusterName
	return mb
}

// WithPhase sets the machine phase.
func (mb *MachineBuilder) WithPhase(phase string) *MachineBuilder {
	mb.phase = phase
	return mb
}

// WithVersion sets the Kubernetes version.
func (mb *MachineBuilder) WithVersion(version string) *MachineBuilder {
	mb.version = version
	return mb
}

// WithProviderID sets the provider ID.
func (mb *MachineBuilder) WithProviderID(providerID string) *MachineBuilder {
	mb.providerID = providerID
	return mb
}

// WithNodeRef sets the NodeRef (makes the machine "ready").
func (mb *MachineBuilder) WithNodeRef(nodeName string) *MachineBuilder {
	mb.nodeRefName = nodeName
	return mb
}

// WithConfigRef sets the Bootstrap.ConfigRef.
func (mb *MachineBuilder) WithConfigRef(kind, name string) *MachineBuilder {
	mb.configRefKind = kind
	mb.configRefName = name
	return mb
}

// WithInfraRef sets the InfrastructureRef.
func (mb *MachineBuilder) WithInfraRef(kind, name string) *MachineBuilder {
	mb.infraRefKind = kind
	mb.infraRefName = name
	return mb
}

// MachineConditionBuilder provides a fluent API for configuring a machine condition.
type MachineConditionBuilder struct {
	machineBuilder *MachineBuilder
	condType       string
	status         corev1.ConditionStatus
	reason         string
	message        string
}

// WithCondition starts configuring a condition for this machine.
func (mb *MachineBuilder) WithCondition(condType string) *MachineConditionBuilder {
	return &MachineConditionBuilder{
		machineBuilder: mb,
		condType:       condType,
	}
}

// True sets the condition status to True.
func (mcb *MachineConditionBuilder) True() *MachineConditionBuilder {
	mcb.status = corev1.ConditionTrue
	return mcb
}

// False sets the condition status to False.
func (mcb *MachineConditionBuilder) False() *MachineConditionBuilder {
	mcb.status = corev1.ConditionFalse
	return mcb
}

// Unknown sets the condition status to Unknown.
func (mcb *MachineConditionBuilder) Unknown() *MachineConditionBuilder {
	mcb.status = corev1.ConditionUnknown
	return mcb
}

// Reason sets the reason for this condition.
func (mcb *MachineConditionBuilder) Reason(reason string) *MachineConditionBuilder {
	mcb.reason = reason
	return mcb
}

// Message sets the message for this condition.
func (mcb *MachineConditionBuilder) Message(message string) *MachineConditionBuilder {
	mcb.message = message
	return mcb
}

// Done returns to the MachineBuilder to continue configuration.
func (mcb *MachineConditionBuilder) Done() *MachineBuilder {
	mcb.machineBuilder.conditions = append(mcb.machineBuilder.conditions, machineCondition{
		condType: mcb.condType,
		status:   mcb.status,
		reason:   mcb.reason,
		message:  mcb.message,
	})
	return mcb.machineBuilder
}

// WithAddress adds an address to the machine status.
func (mb *MachineBuilder) WithAddress(addrType, address string) *MachineBuilder {
	mb.addresses = append(mb.addresses, machineAddress{addrType: addrType, address: address})
	return mb
}

// Create queues the machine creation operation and returns to the harness.
func (mb *MachineBuilder) Create() *Harness {
	mb.harness.t.Helper()
	mb.harness.operations = append(mb.harness.operations, &machineBuilderOp{
		namespace:     mb.namespace,
		name:          mb.name,
		clusterName:   mb.clusterName,
		phase:         mb.phase,
		version:       mb.version,
		providerID:    mb.providerID,
		nodeRefName:   mb.nodeRefName,
		configRefKind: mb.configRefKind,
		configRefName: mb.configRefName,
		infraRefKind:  mb.infraRefKind,
		infraRefName:  mb.infraRefName,
		conditions:    mb.conditions,
		addresses:     mb.addresses,
	})
	return mb.harness
}
