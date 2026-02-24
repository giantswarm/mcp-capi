package harness

// MachineDeploymentBuilder provides a fluent API for building MachineDeployment resources.
type MachineDeploymentBuilder struct {
	harness           *Harness
	namespace         string
	name              string
	clusterName       string
	replicas          int
	nilReplicas       bool // if true, Spec.Replicas is nil (overrides replicas field)
	version           string
	phase             string
	hasStatus         bool // explicit flag to trigger status update even with zero values
	statusReplicas    int
	readyReplicas     int
	updatedReplicas   int
	availableReplicas int
}

// MachineDeployment starts a new MachineDeployment builder.
func (h *Harness) MachineDeployment(namespace, name string) *MachineDeploymentBuilder {
	return &MachineDeploymentBuilder{
		harness:   h,
		namespace: namespace,
		name:      name,
		replicas:  1,
	}
}

// ForCluster sets the cluster name for this MachineDeployment.
func (mdb *MachineDeploymentBuilder) ForCluster(clusterName string) *MachineDeploymentBuilder {
	mdb.clusterName = clusterName
	return mdb
}

// WithReplicas sets the desired replica count.
func (mdb *MachineDeploymentBuilder) WithReplicas(replicas int) *MachineDeploymentBuilder {
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
func (mdb *MachineDeploymentBuilder) WithStatus(total, ready, updated, available int) *MachineDeploymentBuilder {
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
		namespace:         mdb.namespace,
		name:              mdb.name,
		clusterName:       mdb.clusterName,
		replicas:          mdb.replicas,
		nilReplicas:       mdb.nilReplicas,
		version:           mdb.version,
		phase:             mdb.phase,
		hasStatus:         mdb.hasStatus,
		statusReplicas:    mdb.statusReplicas,
		readyReplicas:     mdb.readyReplicas,
		updatedReplicas:   mdb.updatedReplicas,
		availableReplicas: mdb.availableReplicas,
	})
	return mdb.harness
}
