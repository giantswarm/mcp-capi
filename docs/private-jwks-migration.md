# Private-address JWKS settings migration

Use `MCP_OAUTH_ALLOW_PRIVATE_IP_JWKS=true` or Helm `oauth.allowPrivateIPJWKS: true`
only when the Dex issuer host resolves to a private address and mcp-capi
validates forwarded ID tokens (`OAUTH_TRUSTED_AUDIENCES`). It lifts the SSRF
guard of that JWKS fetch for the Dex issuer host only. HTTPS and certificate
verification remain required. Dex discovery and token requests already reach
private addresses and honor `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY`; on a
proxied installation, list the Dex host in `NO_PROXY` when Dex must be reached
directly. `DEX_CA_FILE` adds the CA that signed Dex's certificate to the system
roots for these calls and for the JWKS fetch; it changes no routing.

`MCP_OAUTH_ALLOW_PRIVATE_URLS` and Helm `oauth.allowPrivateURLs` are deprecated
aliases of the JWKS setting: `true` in either permits the private JWKS address,
and startup logs a deprecation warning naming the variable. Invalid boolean
values fail startup.

v0.8.0 (planned for 2027-02-17) refuses the aliases. Replace them with the JWKS
setting before upgrading.
