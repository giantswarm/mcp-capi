# mcp-capi

A Helm chart for mcp-capi - Model Context Protocol server for Cluster API

**Homepage:** <https://github.com/giantswarm/mcp-capi>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Giant Swarm | <team-planeteers@giantswarm.io> |  |

## Source Code

* <https://github.com/giantswarm/mcp-capi>

## Rolling on credential rotation

The server reads its OAuth credentials (the Dex or Google client secret, the
token encryption key, the Valkey password) from a Secret at start and never
again. The pod template carries a `checksum/oauth-secret` annotation so a
changed credential restarts the server:

- With `oauth.existingSecret` (and `global.identity.existingSecret`) empty the
  chart renders the Secret from `oauth.dex.clientSecret` (or
  `oauth.google.clientSecret`), `oauth.encryptionKey` and
  `oauth.storage.valkey.password`; the annotation is the SHA-256 of that
  Secret's data and follows every change of those values.
- With an existing Secret the chart cannot read it; the annotation is
  `oauth.existingSecretChecksum` verbatim. Change it in the same change that
  rotates the Secret (a hash over the new data, a counter, a date). A Flux
  `HelmRelease` can instead feed the Secret's keys into the values above through
  `valuesFrom` entries with `targetPath`, so the chart renders the Secret itself
  and the checksum follows the rotation on its own.
- A Valkey password in its own Secret (`oauth.storage.valkey.existingSecret`)
  is marked the same way by `oauth.storage.valkey.existingSecretChecksum`,
  rendered as `checksum/valkey-secret`.

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| global | object | `{}` |  |
| replicaCount | int | `1` |  |
| image.registry | string | `"gsoci.azurecr.io"` |  |
| image.repository | string | `"giantswarm/mcp-capi"` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| nameOverride | string | `""` |  |
| fullnameOverride | string | `""` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.automount | bool | `false` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.name | string | `""` |  |
| podAnnotations | object | `{}` |  |
| podLabels | object | `{}` |  |
| podSecurityContext.runAsNonRoot | bool | `true` |  |
| podSecurityContext.runAsUser | int | `1000` |  |
| podSecurityContext.runAsGroup | int | `1000` |  |
| podSecurityContext.fsGroup | int | `1000` |  |
| podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| securityContext.runAsNonRoot | bool | `true` |  |
| securityContext.runAsUser | int | `1000` |  |
| securityContext.runAsGroup | int | `1000` |  |
| securityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| service.type | string | `"ClusterIP"` |  |
| service.port | int | `8080` |  |
| resources.limits.memory | string | `"256Mi"` |  |
| resources.requests.cpu | string | `"100m"` |  |
| resources.requests.memory | string | `"128Mi"` |  |
| volumes | list | `[]` |  |
| volumeMounts | list | `[]` |  |
| nodeSelector | object | `{}` |  |
| tolerations | list | `[]` |  |
| affinity | object | `{}` |  |
| readOnly | bool | `true` | Register only the tools that read (list, get, inspect, export) and refuse every mutating Kubernetes call. Set to false to offer the create, scale, upgrade, pause, resume and delete tools; the person's RBAC still applies. |
| gitopsGuard | bool | `true` | Refuse writes to objects owned by a GitOps controller (Flux Kustomization or HelmRelease, Argo CD Application) or a Helm release: the change would be reverted on the next reconciliation and belongs in Git. Only matters when readOnly is false. An object labelled giantswarm.io/prevent-deletion is never deleted, whatever the policy. |
| exposeKubeconfig | bool | `false` | Offer capi_get_kubeconfig and let capi_backup_cluster include Secrets: the workload cluster's admin kubeconfig leaves the server. Off by default and independent of readOnly; without it the tool is not registered and the server refuses the export. Never enable this on a Giant Swarm management cluster. |
| oauth.enabled | bool | `false` |  |
| oauth.baseURL | string | `""` |  |
| oauth.provider | string | `"dex"` |  |
| oauth.dex.issuerURL | string | `""` |  |
| oauth.dex.clientID | string | `""` |  |
| oauth.dex.clientSecret | string | `""` |  |
| oauth.dex.kubernetesAuthenticatorClientID | string | `""` |  |
| oauth.dex.caSecret.name | string | `""` |  |
| oauth.dex.caSecret.key | string | `"ca.crt"` |  |
| oauth.google.clientID | string | `""` |  |
| oauth.google.clientSecret | string | `""` |  |
| oauth.existingSecret | string | `""` |  |
| oauth.existingSecretChecksum | string | `""` |  |
| oauth.encryptionKey | string | `""` |  |
| oauth.allowPublicRegistration | bool | `false` |  |
| oauth.allowPrivateURLs | bool | `false` |  |
| oauth.trustedAudiences | list | `[]` |  |
| oauth.downstream.enabled | bool | `true` |  |
| oauth.storage.type | string | `"memory"` |  |
| oauth.storage.valkey.url | string | `""` |  |
| oauth.storage.valkey.password | string | `""` |  |
| oauth.storage.valkey.tls.enabled | bool | `false` |  |
| oauth.storage.valkey.keyPrefix | string | `"mcp:"` |  |
| oauth.storage.valkey.existingSecret | string | `""` |  |
| oauth.storage.valkey.existingSecretChecksum | string | `""` |  |
| oauth.storage.valkey.secretKeyPassword | string | `"valkey-password"` |  |
| gatewayAPI.enabled | bool | `false` |  |
| gatewayAPI.httpRoute.parentRefs | list | `[]` |  |
| gatewayAPI.httpRoute.hostnames | list | `[]` |  |
| gatewayAPI.httpRoute.labels | object | `{}` |  |
| gatewayAPI.httpRoute.annotations | object | `{}` |  |
| gatewayAPI.backendTrafficPolicy.enabled | bool | `false` |  |
| gatewayAPI.backendTrafficPolicy.timeout | string | `"0s"` |  |
| gatewayAPI.backendTrafficPolicy.labels | object | `{}` |  |
| gatewayAPI.backendTrafficPolicy.annotations | object | `{}` |  |
| ciliumNetworkPolicy.enabled | bool | `true` |  |
| ciliumNetworkPolicy.labels | object | `{}` |  |
| ciliumNetworkPolicy.annotations | object | `{}` |  |
