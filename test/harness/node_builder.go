package harness

// nodeAddress holds a node address configuration.
type nodeAddress struct {
	addrType string
	address  string
}

// nodeTaint holds a node taint configuration.
type nodeTaint struct {
	key    string
	value  string
	effect string
}

// nodeResources holds resource quantities for a node.
type nodeResources struct {
	cpu    string
	memory string
	pods   string
}

// nodeInfoConfig holds node system information.
type nodeInfoConfig struct {
	osName                  string
	osImage                 string
	kernelVersion           string
	containerRuntimeVersion string
	kubeletVersion          string
	architecture            string
}

// NodeBuilder provides a fluent API for building Kubernetes Node resources with custom properties.
type NodeBuilder struct {
	harness *Harness
	nodeCreateOptions
}

// Node starts a new node builder with sensible defaults.
func (h *Harness) Node(name string) *NodeBuilder {
	return &NodeBuilder{
		harness: h,
		nodeCreateOptions: nodeCreateOptions{
			name: name,
			nodeInfo: nodeInfoConfig{
				osName:                  "linux",
				osImage:                 "Ubuntu 22.04.3 LTS",
				kernelVersion:           "5.15.0-91-generic",
				containerRuntimeVersion: "containerd://1.7.2",
				kubeletVersion:          "v1.29.0",
				architecture:            "amd64",
			},
			capacity: nodeResources{
				cpu:    "4",
				memory: "8Gi",
				pods:   "110",
			},
			allocatable: nodeResources{
				cpu:    "3500m",
				memory: "7Gi",
				pods:   "110",
			},
		},
	}
}

// WithProviderID sets the provider ID on the node.
func (nb *NodeBuilder) WithProviderID(providerID string) *NodeBuilder {
	nb.providerID = providerID
	return nb
}

// WithUnschedulable sets the node as unschedulable (cordoned).
func (nb *NodeBuilder) WithUnschedulable(unschedulable bool) *NodeBuilder {
	nb.unschedulable = unschedulable
	return nb
}

// WithCondition starts configuring a condition for this node.
func (nb *NodeBuilder) WithCondition(condType string) *simpleConditionBuilder[*NodeBuilder] {
	return &simpleConditionBuilder[*NodeBuilder]{
		condType: condType,
		done: func(c simpleCondition) *NodeBuilder {
			nb.conditions = append(nb.conditions, c)
			return nb
		},
	}
}

// WithAddress adds an address to the node.
func (nb *NodeBuilder) WithAddress(addrType, address string) *NodeBuilder {
	nb.addresses = append(nb.addresses, nodeAddress{addrType: addrType, address: address})
	return nb
}

// WithTaint adds a taint to the node.
func (nb *NodeBuilder) WithTaint(key, value, effect string) *NodeBuilder {
	nb.taints = append(nb.taints, nodeTaint{key: key, value: value, effect: effect})
	return nb
}

// WithKubeletVersion sets the kubelet version.
func (nb *NodeBuilder) WithKubeletVersion(version string) *NodeBuilder {
	nb.nodeInfo.kubeletVersion = version
	return nb
}

// Create queues the node creation operation and returns to the harness.
func (nb *NodeBuilder) Create() *Harness {
	nb.harness.t.Helper()
	nb.harness.operations = append(nb.harness.operations, &nodeBuilderOp{
		nodeCreateOptions: nb.nodeCreateOptions,
	})
	return nb.harness
}
