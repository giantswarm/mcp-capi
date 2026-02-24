package harness

// MachineDeploymentBuilder provides a fluent API for building MachineDeployment resources.
type MachineDeploymentBuilder struct {
	harness *Harness
	machineDeploymentCreateOptions
}

// MachineDeployment starts a new MachineDeployment builder.
func (h *Harness) MachineDeployment(namespace, name string) *MachineDeploymentBuilder {
	return &MachineDeploymentBuilder{
		harness: h,
		machineDeploymentCreateOptions: machineDeploymentCreateOptions{
			namespace: namespace,
			name:      name,
			replicas:  1,
		},
	}
}

// ForCluster sets the cluster name for this MachineDeployment.
func (mdb *MachineDeploymentBuilder) ForCluster(clusterName string) *MachineDeploymentBuilder {
	mdb.clusterName = clusterName
	return mdb
}

// WithReplicas sets the desired replica count.
func (mdb *MachineDeploymentBuilder) WithReplicas(replicas int32) *MachineDeploymentBuilder {
	mdb.replicas = replicas
	return mdb
}

// WithNilReplicas sets Spec.Replicas to nil (no desired replica count).
func (mdb *MachineDeploymentBuilder) WithNilReplicas() *MachineDeploymentBuilder {
	mdb.nilReplicas = true
	return mdb
}

// WithVersion sets the Kubernetes version.
func (mdb *MachineDeploymentBuilder) WithVersion(version string) *MachineDeploymentBuilder {
	mdb.version = version
	return mdb
}

// WithPhase sets the phase on the MachineDeployment status.
func (mdb *MachineDeploymentBuilder) WithPhase(phase string) *MachineDeploymentBuilder {
	mdb.phase = phase
	return mdb
}

// WithStatus sets the status replica counts.
// Setting status explicitly triggers a status update even when all values are zero.
func (mdb *MachineDeploymentBuilder) WithStatus(total, ready, updated, available int32) *MachineDeploymentBuilder {
	mdb.hasStatus = true
	mdb.statusReplicas = total
	mdb.readyReplicas = ready
	mdb.updatedReplicas = updated
	mdb.availableReplicas = available
	return mdb
}

// Create queues the MachineDeployment creation and returns to the harness.
func (mdb *MachineDeploymentBuilder) Create() *Harness {
	mdb.harness.t.Helper()
	mdb.harness.operations = append(mdb.harness.operations, &machineDeploymentOp{
		machineDeploymentCreateOptions: mdb.machineDeploymentCreateOptions,
	})
	return mdb.harness
}
