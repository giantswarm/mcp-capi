package harness

import (
	corev1 "k8s.io/api/core/v1"
)

// nodeCondition holds a node condition configuration.
type nodeCondition struct {
	condType string
	status   corev1.ConditionStatus
	reason   string
	message  string
}

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
	os                      string
	osImage                 string
	kernelVersion           string
	containerRuntimeVersion string
	kubeletVersion          string
	architecture            string
}

// NodeBuilder provides a fluent API for building Kubernetes Node resources with custom properties.
type NodeBuilder struct {
	harness       *Harness
	name          string
	providerID    string
	unschedulable bool
	conditions    []nodeCondition
	addresses     []nodeAddress
	taints        []nodeTaint
	capacity      nodeResources
	allocatable   nodeResources
	nodeInfo      nodeInfoConfig
}

// Node starts a new node builder with sensible defaults.
func (h *Harness) Node(name string) *NodeBuilder {
	return &NodeBuilder{
		harness: h,
		name:    name,
		nodeInfo: nodeInfoConfig{
			os:                      "linux",
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

// NodeConditionBuilder provides a fluent API for configuring a node condition.
type NodeConditionBuilder struct {
	nodeBuilder *NodeBuilder
	condType    string
	status      corev1.ConditionStatus
	reason      string
	message     string
}

// WithCondition starts configuring a condition for this node.
func (nb *NodeBuilder) WithCondition(condType string) *NodeConditionBuilder {
	return &NodeConditionBuilder{
		nodeBuilder: nb,
		condType:    condType,
	}
}

// True sets the condition status to True.
func (ncb *NodeConditionBuilder) True() *NodeConditionBuilder {
	ncb.status = corev1.ConditionTrue
	return ncb
}

// False sets the condition status to False.
func (ncb *NodeConditionBuilder) False() *NodeConditionBuilder {
	ncb.status = corev1.ConditionFalse
	return ncb
}

// Unknown sets the condition status to Unknown.
func (ncb *NodeConditionBuilder) Unknown() *NodeConditionBuilder {
	ncb.status = corev1.ConditionUnknown
	return ncb
}

// Reason sets the reason for this condition.
func (ncb *NodeConditionBuilder) Reason(reason string) *NodeConditionBuilder {
	ncb.reason = reason
	return ncb
}

// Message sets the message for this condition.
func (ncb *NodeConditionBuilder) Message(message string) *NodeConditionBuilder {
	ncb.message = message
	return ncb
}

// Done returns to the NodeBuilder to continue configuration.
func (ncb *NodeConditionBuilder) Done() *NodeBuilder {
	ncb.nodeBuilder.conditions = append(ncb.nodeBuilder.conditions, nodeCondition{
		condType: ncb.condType,
		status:   ncb.status,
		reason:   ncb.reason,
		message:  ncb.message,
	})
	return ncb.nodeBuilder
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
		name:          nb.name,
		providerID:    nb.providerID,
		unschedulable: nb.unschedulable,
		conditions:    nb.conditions,
		addresses:     nb.addresses,
		taints:        nb.taints,
		capacity:      nb.capacity,
		allocatable:   nb.allocatable,
		nodeInfo:      nb.nodeInfo,
	})
	return nb.harness
}
