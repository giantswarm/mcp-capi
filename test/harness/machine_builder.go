package harness

// machineAddress holds a machine address configuration.
type machineAddress struct {
	addrType string
	address  string
}

// MachineBuilder provides a fluent API for building individual CAPI Machine resources.
// Use this for fine-grained control over machine fields like Bootstrap.ConfigRef,
// InfrastructureRef, Version, and Conditions.
type MachineBuilder struct {
	harness *Harness
	machineCustomCreateOptions
}

// Machine starts a new machine builder.
func (h *Harness) Machine(namespace, name string) *MachineBuilder {
	return &MachineBuilder{
		harness: h,
		machineCustomCreateOptions: machineCustomCreateOptions{
			namespace: namespace,
			name:      name,
		},
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

// WithCondition starts configuring a condition for this machine.
func (mb *MachineBuilder) WithCondition(condType string) *simpleConditionBuilder[*MachineBuilder] {
	return &simpleConditionBuilder[*MachineBuilder]{
		condType: condType,
		done: func(c simpleCondition) *MachineBuilder {
			mb.conditions = append(mb.conditions, c)
			return mb
		},
	}
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
		machineCustomCreateOptions: mb.machineCustomCreateOptions,
	})
	return mb.harness
}
