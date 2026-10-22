# The Go binary is built by CircleCI (architect/go-build) and attached to the
# build context as <binary>-<os>-<arch>; this image only assembles the runtime.
# For a local build, produce the binary first:
#   CGO_ENABLED=0 go build -o mcp-capi-linux-amd64 .
#
# The certs stage runs on the build platform (its files are arch-independent):
# the current CA bundle from alpine's ca-certificates, and the giantswarm user
# (UID/GID 1000, as the chart's security context sets) that the scratch image
# names.
FROM --platform=$BUILDPLATFORM gsoci.azurecr.io/giantswarm/alpine:3.24.3 AS certs
RUN apk add --no-cache ca-certificates \
 && addgroup -g 1000 giantswarm \
 && adduser -D -H -u 1000 -G giantswarm -s /sbin/nologin giantswarm

FROM scratch

COPY --from=certs /etc/passwd /etc/passwd
COPY --from=certs /etc/group /etc/group
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ARG TARGETOS
ARG TARGETARCH
COPY mcp-capi-${TARGETOS}-${TARGETARCH} /mcp-capi
USER giantswarm

ENTRYPOINT ["/mcp-capi"]
