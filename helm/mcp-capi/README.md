# mcp-capi

![Version: 0.1.0](https://img.shields.io/badge/Version-0.1.0-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.1.0](https://img.shields.io/badge/AppVersion-0.1.0-informational?style=flat-square)

A Helm chart for mcp-capi - Model Context Protocol server for Cluster API

**Homepage:** <https://github.com/giantswarm/mcp-capi>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Giant Swarm | <team-planeteers@giantswarm.io> |  |

## Source Code

* <https://github.com/giantswarm/mcp-capi>

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
