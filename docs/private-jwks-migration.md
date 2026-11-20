# Private-address JWKS settings migration

`MCP_OAUTH_ALLOW_PRIVATE_IP_JWKS` and Helm `oauth.allowPrivateIPJWKS` have no effect:
the configured Dex issuer host may resolve to a private address without them when
mcp-capi validates forwarded ID tokens (`OAUTH_TRUSTED_AUDIENCES`). HTTPS and
certificate verification remain required. Setting the key to `true` logs a warning
at startup; remove it. Dex discovery and token requests honor `HTTP_PROXY`,
`HTTPS_PROXY` and `NO_PROXY`; on a proxied installation, list the Dex host in
`NO_PROXY` when Dex must be reached directly. `DEX_CA_FILE` adds the CA that signed
Dex's certificate to the system roots for these calls and for the JWKS fetch; it
changes no routing.

`MCP_OAUTH_ALLOW_PRIVATE_URLS` and Helm `oauth.allowPrivateURLs` are deprecated
aliases of the same setting and equally have no effect. Startup logs a deprecation
warning naming the variable. Invalid boolean values fail startup.

v0.8.0 (planned for 2027-02-17) refuses the aliases. Remove them before upgrading.
