# Machine and Node Management Tools

This document describes the MCP tools available for managing CAPI machines, machine deployments, machine sets, and Kubernetes nodes.

## Machine Operations

### capi_list_machines
List all machines in a namespace with optional filtering by cluster.

**Parameters:**
- `namespace` (required): Namespace to list machines from
- `clusterName` (optional): Filter machines by cluster name

**Example:**
```
capi_list_machines --namespace default --clusterName my-cluster
```

### capi_get_machine
Get detailed information about a specific machine.

**Parameters:**
- `namespace` (required): Machine namespace
- `name` (required): Machine name

**Example:**
```
capi_get_machine --namespace default --name my-cluster-control-plane-abc123
```

### capi_delete_machine
Delete a specific machine. The machine will be drained first if it has an associated node.

**Parameters:**
- `namespace` (required): Machine namespace
- `name` (required): Machine name
- `force` (optional): Force deletion even if machine is healthy

**Example:**
```
capi_delete_machine --namespace default --name worker-node-xyz --force
```

### capi_remediate_machine
Trigger machine health check remediation for a machine.

**Parameters:**
- `namespace` (required): Machine namespace
- `name` (required): Machine name

**Example:**
```
capi_remediate_machine --namespace default --name unhealthy-worker-123
```

## MachineDeployment Operations

### capi_create_machinedeployment
Create a new worker node pool (MachineDeployment).

**Parameters:**
- `namespace` (required): Namespace for the machine deployment
- `name` (required): Name of the machine deployment
- `cluster_name` (required): Name of the cluster
- `replicas` (optional): Number of replicas (default: 1)
- `version` (optional): Kubernetes version
- `infra_kind` (required): Infrastructure template kind (e.g., AWSMachineTemplate)
- `infra_name` (required): Infrastructure template name
- `infra_api_version` (optional): Infrastructure template API version
- `bootstrap_kind` (required): Bootstrap config kind (e.g., KubeadmConfigTemplate)
- `bootstrap_name` (required): Bootstrap config template name
- `bootstrap_api_version` (optional): Bootstrap config API version

**Example:**
```
capi_create_machinedeployment --namespace default --name worker-pool-1 \
  --cluster_name my-cluster --replicas 3 --version v1.29.0 \
  --infra_kind AWSMachineTemplate --infra_name aws-worker-template \
  --bootstrap_kind KubeadmConfigTemplate --bootstrap_name kubeadm-worker-config
```

### capi_list_machinedeployments
List all machine deployments in a namespace.

**Parameters:**
- `namespace` (required): Namespace to list from
- `clusterName` (optional): Filter by cluster name

**Example:**
```
capi_list_machinedeployments --namespace default
```

### capi_scale_machinedeployment
Scale worker nodes up or down.

**Parameters:**
- `namespace` (required): MachineDeployment namespace
- `name` (required): MachineDeployment name
- `replicas` (required): Number of replicas to scale to

**Example:**
```
capi_scale_machinedeployment --namespace default --name worker-pool-1 --replicas 5
```

### capi_update_machinedeployment
Update MachineDeployment configuration.

**Parameters:**
- `namespace` (required): MachineDeployment namespace
- `name` (required): MachineDeployment name
- `version` (optional): Kubernetes version to update to
- `replicas` (optional): Number of replicas
- `min_ready_seconds` (optional): Minimum ready seconds
- `labels` (optional): Labels to add/update (empty value removes label)
- `annotations` (optional): Annotations to add/update

**Example:**
```
capi_update_machinedeployment --namespace default --name worker-pool-1 \
  --version v1.29.1 --labels '{"environment": "production", "team": "platform"}'
```

### capi_rollout_machinedeployment
Trigger a rolling update of a MachineDeployment.

**Parameters:**
- `namespace` (required): MachineDeployment namespace
- `name` (required): MachineDeployment name
- `reason` (optional): Reason for the rollout

**Example:**
```
capi_rollout_machinedeployment --namespace default --name worker-pool-1 \
  --reason "Apply security patches"
```

## MachineSet Operations

### capi_list_machinesets
List all MachineSets in a namespace.

**Parameters:**
- `namespace` (required): Namespace to list machine sets in
- `clusterName` (optional): Filter by cluster name

**Example:**
```
capi_list_machinesets --namespace default --clusterName my-cluster
```

### capi_get_machineset
Get detailed information about a specific MachineSet.

**Parameters:**
- `namespace` (required): MachineSet namespace
- `name` (required): MachineSet name

**Example:**
```
capi_get_machineset --namespace default --name worker-pool-1-abc123
```

## Node Operations

### capi_drain_node
Safely drain a Kubernetes node, evicting all pods except DaemonSets if requested.

**Parameters:**
- `namespace` (optional): Machine namespace (required if using machine_name)
- `machine_name` (optional): Machine name to get node from
- `node_name` (optional): Node name to drain directly
- `ignore_daemonsets` (optional): Ignore DaemonSet-managed pods
- `delete_local_data` (optional): Delete pods with local storage
- `force` (optional): Force deletion of pods
- `grace_period_seconds` (optional): Grace period for pod termination

**Note:** Either `node_name` or both `namespace` and `machine_name` must be provided.

**Example:**
```
# Drain by node name
capi_drain_node --node_name worker-node-1 --ignore_daemonsets --delete_local_data

# Drain by machine reference
capi_drain_node --namespace default --machine_name my-cluster-worker-abc123
```

### capi_cordon_node
Cordon (make unschedulable) or uncordon a Kubernetes node.

**Parameters:**
- `namespace` (optional): Machine namespace (required if using machine_name)
- `machine_name` (optional): Machine name to get node from
- `node_name` (optional): Node name to cordon/uncordon directly
- `uncordon` (optional): Set to true to uncordon (make schedulable)

**Example:**
```
# Cordon a node
capi_cordon_node --node_name worker-node-1

# Uncordon a node
capi_cordon_node --node_name worker-node-1 --uncordon
```

### capi_node_status
Get detailed status information about a node in the workload cluster.

**Parameters:**
- `namespace` (optional): Machine namespace (required if using machine_name)
- `machine_name` (optional): Machine name to get node from
- `node_name` (optional): Node name to get status for directly

**Example:**
```
capi_node_status --namespace default --machine_name my-cluster-worker-abc123
```

## Workflow Examples

### Scaling Worker Nodes
```bash
# Check current worker pools
capi_list_machinedeployments --namespace default

# Scale up workers
capi_scale_machinedeployment --namespace default --name worker-pool-1 --replicas 5

# Monitor scaling progress
capi_list_machines --namespace default --clusterName my-cluster
```

### Rolling Update
```bash
# Update Kubernetes version
capi_update_machinedeployment --namespace default --name worker-pool-1 --version v1.29.1

# Trigger rollout
capi_rollout_machinedeployment --namespace default --name worker-pool-1 --reason "Kubernetes upgrade"

# Monitor rollout
capi_list_machinesets --namespace default
```

### Node Maintenance
```bash
# Cordon node to prevent new workloads
capi_cordon_node --node_name worker-node-1

# Drain node to move workloads
capi_drain_node --node_name worker-node-1 --ignore_daemonsets --delete_local_data

# Perform maintenance...

# Uncordon node when ready
capi_cordon_node --node_name worker-node-1 --uncordon
```

## Important Notes

1. **Machine Deletion**: When deleting machines, the associated node will be drained first to ensure workloads are safely moved.

2. **Scaling Operations**: When scaling down, CAPI will automatically select machines to delete based on various factors including machine age and health.

3. **Rolling Updates**: Updates respect the MachineDeployment's update strategy and will maintain availability during the rollout.

4. **Node Draining**: The current implementation only cordons nodes. Full pod eviction is not yet implemented - use kubectl drain for complete functionality.

5. **Management vs Workload Cluster**: Most operations work on the management cluster. Node operations that require workload cluster access are limited. 