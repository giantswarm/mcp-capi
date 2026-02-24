package harness

import (
	corev1 "k8s.io/api/core/v1"
)

// machineSetCondition holds a MachineSet condition configuration.
type machineSetCondition struct {
	condType string
	status   corev1.ConditionStatus
	reason   string
	message  string
}

// MachineSetBuilder provides a fluent API for building MachineSet resources.
type MachineSetBuilder struct {
	harness *Harness
	machineSetCreateOptions
}

// MachineSet starts a new MachineSet builder.
func (h *Harness) MachineSet(namespace, name string) *MachineSetBuilder {
	return &MachineSetBuilder{
		harness: h,
		machineSetCreateOptions: machineSetCreateOptions{
			namespace: namespace,
			name:      name,
			replicas:  1,
		},
	}
}

// ForCluster sets the cluster name for this MachineSet.
func (msb *MachineSetBuilder) ForCluster(clusterName string) *MachineSetBuilder {
	msb.clusterName = clusterName
	return msb
}

// WithReplicas sets the desired replica count.
func (msb *MachineSetBuilder) WithReplicas(replicas int32) *MachineSetBuilder {
	msb.replicas = replicas
	return msb
}

// WithVersion sets the Kubernetes version.
func (msb *MachineSetBuilder) WithVersion(version string) *MachineSetBuilder {
	msb.version = version
	return msb
}

// WithStatus sets the status replica counts.
// Setting status explicitly triggers a status update even when all values are zero.
func (msb *MachineSetBuilder) WithStatus(total, ready, available int) *MachineSetBuilder {
	msb.hasStatus = true
	msb.statusReplicas = total
	msb.readyReplicas = ready
	msb.availableReplicas = available
	return msb
}

// WithInfraRef sets the infrastructure reference.
func (msb *MachineSetBuilder) WithInfraRef(kind, name string) *MachineSetBuilder {
	msb.infraRefKind = kind
	msb.infraRefName = name
	return msb
}

// WithBootstrapRef sets the bootstrap config reference.
func (msb *MachineSetBuilder) WithBootstrapRef(kind, name string) *MachineSetBuilder {
	msb.bootstrapKind = kind
	msb.bootstrapName = name
	return msb
}

// OwnedBy sets the owner MachineDeployment name.
func (msb *MachineSetBuilder) OwnedBy(mdName string) *MachineSetBuilder {
	msb.ownerMDName = mdName
	return msb
}

// OwnedByKind sets a custom owner reference with an arbitrary kind and name.
// Use this to test non-MachineDeployment owners.
func (msb *MachineSetBuilder) OwnedByKind(kind, name string) *MachineSetBuilder {
	msb.ownerKind = kind
	msb.ownerName = name
	return msb
}

// WithNilReplicas sets Spec.Replicas to nil (no desired replica count).
func (msb *MachineSetBuilder) WithNilReplicas() *MachineSetBuilder {
	msb.nilReplicas = true
	return msb
}

// WithFailureReason sets the failure reason on the MachineSet status.
func (msb *MachineSetBuilder) WithFailureReason(reason string) *MachineSetBuilder {
	msb.failureReason = reason
	return msb
}

// WithFailureMessage sets the failure message on the MachineSet status.
func (msb *MachineSetBuilder) WithFailureMessage(message string) *MachineSetBuilder {
	msb.failureMessage = message
	return msb
}

// MachineSetConditionBuilder provides a fluent API for configuring a MachineSet condition.
type MachineSetConditionBuilder struct {
	machineSetBuilder *MachineSetBuilder
	condType          string
	status            corev1.ConditionStatus
	reason            string
	message           string
}

// WithCondition starts configuring a condition for this MachineSet.
func (msb *MachineSetBuilder) WithCondition(condType string) *MachineSetConditionBuilder {
	return &MachineSetConditionBuilder{
		machineSetBuilder: msb,
		condType:          condType,
	}
}

// True sets the condition status to True.
func (mscb *MachineSetConditionBuilder) True() *MachineSetConditionBuilder {
	mscb.status = corev1.ConditionTrue
	return mscb
}

// False sets the condition status to False.
func (mscb *MachineSetConditionBuilder) False() *MachineSetConditionBuilder {
	mscb.status = corev1.ConditionFalse
	return mscb
}

// Unknown sets the condition status to Unknown.
func (mscb *MachineSetConditionBuilder) Unknown() *MachineSetConditionBuilder {
	mscb.status = corev1.ConditionUnknown
	return mscb
}

// Reason sets the reason for this condition.
func (mscb *MachineSetConditionBuilder) Reason(reason string) *MachineSetConditionBuilder {
	mscb.reason = reason
	return mscb
}

// Message sets the message for this condition.
func (mscb *MachineSetConditionBuilder) Message(message string) *MachineSetConditionBuilder {
	mscb.message = message
	return mscb
}

// Done returns to the MachineSetBuilder to continue configuration.
func (mscb *MachineSetConditionBuilder) Done() *MachineSetBuilder {
	mscb.machineSetBuilder.conditions = append(mscb.machineSetBuilder.conditions, machineSetCondition{
		condType: mscb.condType,
		status:   mscb.status,
		reason:   mscb.reason,
		message:  mscb.message,
	})
	return mscb.machineSetBuilder
}

// Create queues the MachineSet creation and returns to the harness.
func (msb *MachineSetBuilder) Create() *Harness {
	msb.harness.t.Helper()
	msb.harness.operations = append(msb.harness.operations, &machineSetOp{
		machineSetCreateOptions: msb.machineSetCreateOptions,
	})
	return msb.harness
}
