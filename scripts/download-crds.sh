#!/usr/bin/env bash
#
# download-crds.sh - Download Cluster API CRDs from upstream repositories
#
# Usage:
#   ./scripts/download-crds.sh --all           # Download all provider CRDs
#   ./scripts/download-crds.sh capi capa       # Download specific providers
#   ./scripts/download-crds.sh --help          # Show help
#
# Environment Variables:
#   CAPI_VERSION    - Cluster API version (default: v1.11.2)
#   CAPA_VERSION    - AWS provider version (default: v2.9.2)
#   CAPZ_VERSION    - Azure provider version (default: v1.21.1)
#   CAPV_VERSION    - vSphere provider version (default: v1.14.0)
#   CAPVCD_VERSION  - Cloud Director provider version (default: v1.3.2)
#   CAPG_VERSION    - GCP provider version (default: v1.10.0)
#

set -euo pipefail

# =============================================================================
# Constants
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
readonly PROJECT_ROOT
readonly OUTPUT_DIR="${PROJECT_ROOT}/crds"
readonly GITHUB_RAW_URL="https://raw.githubusercontent.com"

# API group prefixes
readonly API_GROUP_CORE="cluster.x-k8s.io"
readonly API_GROUP_CONTROLPLANE="controlplane.cluster.x-k8s.io"
readonly API_GROUP_INFRA="infrastructure.cluster.x-k8s.io"

# Available providers
readonly ALL_PROVIDERS="capi capa capz capv capvcd capg"

# =============================================================================
# Default Versions (can be overridden by environment variables)
# =============================================================================

CAPI_VERSION="${CAPI_VERSION:-v1.11.2}"
CAPA_VERSION="${CAPA_VERSION:-v2.9.2}"
CAPZ_VERSION="${CAPZ_VERSION:-v1.21.1}"
CAPV_VERSION="${CAPV_VERSION:-v1.14.0}"
CAPVCD_VERSION="${CAPVCD_VERSION:-v1.3.2}"
CAPG_VERSION="${CAPG_VERSION:-v1.10.0}"

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo "$*"
}

log_error() {
    echo "ERROR: $*" >&2
}

# Download a single CRD file
# Arguments:
#   $1 - Base URL for the CRD
#   $2 - Output filename
download_crd() {
    local url="$1"
    local filename="$2"
    local output_path="${OUTPUT_DIR}/${filename}"

    log_info "  - ${filename}"
    if ! curl -sfL "${url}" -o "${output_path}"; then
        log_error "Failed to download ${filename} from ${url}"
        return 1
    fi
}

# Download multiple CRDs for a provider
# Arguments:
#   $1 - Base URL for CRDs
#   $2 - API group (e.g., cluster.x-k8s.io)
#   $3+ - CRD names (e.g., clusters machines)
download_provider_crds() {
    local base_url="$1"
    local api_group="$2"
    shift 2
    local crds=("$@")

    for crd in "${crds[@]}"; do
        local filename="${api_group}_${crd}.yaml"
        download_crd "${base_url}/${filename}" "${filename}"
    done
}

# =============================================================================
# Provider Download Functions
# =============================================================================

download_capi() {
    local version="${CAPI_VERSION}"
    local repo="kubernetes-sigs/cluster-api"
    local base_url="${GITHUB_RAW_URL}/${repo}/${version}/config/crd/bases"

    log_info "Downloading CAPI CRDs from ${version}..."

    # Core cluster API CRDs
    local core_crds=(
        clusters
        machines
        machinedeployments
        machinesets
        machinepools
        clusterclasses
        machinehealthchecks
    )
    download_provider_crds "${base_url}" "${API_GROUP_CORE}" "${core_crds[@]}"

    # KubeadmControlPlane CRD (from different path)
    local kcp_url="${GITHUB_RAW_URL}/${repo}/${version}/controlplane/kubeadm/config/crd/bases"
    local kcp_filename="${API_GROUP_CONTROLPLANE}_kubeadmcontrolplanes.yaml"
    download_crd "${kcp_url}/${kcp_filename}" "${kcp_filename}"

    log_info "Successfully downloaded CAPI CRDs to crds/"
}

download_capa() {
    local version="${CAPA_VERSION}"
    local repo="kubernetes-sigs/cluster-api-provider-aws"
    local base_url="${GITHUB_RAW_URL}/${repo}/${version}/config/crd/bases"

    log_info "Downloading CAPA CRDs from ${version}..."

    local crds=(
        awsclusters
        awsmachines
        awsmachinetemplates
        awsclustertemplates
        awsmachinepools
    )
    download_provider_crds "${base_url}" "${API_GROUP_INFRA}" "${crds[@]}"

    log_info "Successfully downloaded CAPA CRDs to crds/"
}

download_capz() {
    local version="${CAPZ_VERSION}"
    local repo="kubernetes-sigs/cluster-api-provider-azure"
    local base_url="${GITHUB_RAW_URL}/${repo}/${version}/config/crd/bases"

    log_info "Downloading CAPZ CRDs from ${version}..."

    local crds=(
        azureclusters
        azuremachines
        azuremachinetemplates
        azureclustertemplates
        azuremachinepools
    )
    download_provider_crds "${base_url}" "${API_GROUP_INFRA}" "${crds[@]}"

    log_info "Successfully downloaded CAPZ CRDs to crds/"
}

download_capv() {
    local version="${CAPV_VERSION}"
    local repo="kubernetes-sigs/cluster-api-provider-vsphere"
    # Note: CAPV uses a different path structure
    local base_url="${GITHUB_RAW_URL}/${repo}/${version}/config/default/crd/bases"

    log_info "Downloading CAPV CRDs from ${version}..."

    local crds=(
        vsphereclusters
        vspheremachines
        vspheremachinetemplates
        vsphereclustertemplates
    )
    download_provider_crds "${base_url}" "${API_GROUP_INFRA}" "${crds[@]}"

    log_info "Successfully downloaded CAPV CRDs to crds/"
}

download_capvcd() {
    local version="${CAPVCD_VERSION}"
    # Note: CAPVCD uses vmware org instead of kubernetes-sigs
    local repo="vmware/cluster-api-provider-cloud-director"
    local base_url="${GITHUB_RAW_URL}/${repo}/${version}/config/crd/bases"

    log_info "Downloading CAPVCD CRDs from ${version}..."

    local crds=(
        vcdclusters
        vcdmachines
        vcdmachinetemplates
        vcdclustertemplates
    )
    download_provider_crds "${base_url}" "${API_GROUP_INFRA}" "${crds[@]}"

    log_info "Successfully downloaded CAPVCD CRDs to crds/"
}

download_capg() {
    local version="${CAPG_VERSION}"
    local repo="kubernetes-sigs/cluster-api-provider-gcp"
    local base_url="${GITHUB_RAW_URL}/${repo}/${version}/config/crd/bases"

    log_info "Downloading CAPG CRDs from ${version}..."

    local crds=(
        gcpclusters
        gcpmachines
        gcpmachinetemplates
        gcpclustertemplates
    )
    download_provider_crds "${base_url}" "${API_GROUP_INFRA}" "${crds[@]}"

    log_info "Successfully downloaded CAPG CRDs to crds/"
}

# =============================================================================
# Main Logic
# =============================================================================

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS] [PROVIDERS...]

Download Cluster API CRDs from upstream repositories.

Options:
    --all       Download CRDs for all providers
    --help, -h  Show this help message

Providers:
    capi        Cluster API core CRDs
    capa        AWS provider (CAPA) CRDs
    capz        Azure provider (CAPZ) CRDs
    capv        vSphere provider (CAPV) CRDs
    capvcd      Cloud Director provider (CAPVCD) CRDs
    capg        GCP provider (CAPG) CRDs

Examples:
    $(basename "$0") --all           # Download all provider CRDs
    $(basename "$0") capi capa       # Download CAPI and CAPA CRDs
    $(basename "$0") capz            # Download only Azure CRDs

Environment Variables:
    CAPI_VERSION    Cluster API version (default: ${CAPI_VERSION})
    CAPA_VERSION    AWS provider version (default: ${CAPA_VERSION})
    CAPZ_VERSION    Azure provider version (default: ${CAPZ_VERSION})
    CAPV_VERSION    vSphere provider version (default: ${CAPV_VERSION})
    CAPVCD_VERSION  Cloud Director version (default: ${CAPVCD_VERSION})
    CAPG_VERSION    GCP provider version (default: ${CAPG_VERSION})
EOF
}

download_providers() {
    local providers=("$@")

    # Ensure output directory exists
    mkdir -p "${OUTPUT_DIR}"

    for provider in "${providers[@]}"; do
        case "${provider}" in
            capi)   download_capi ;;
            capa)   download_capa ;;
            capz)   download_capz ;;
            capv)   download_capv ;;
            capvcd) download_capvcd ;;
            capg)   download_capg ;;
            *)
                log_error "Unknown provider: ${provider}"
                log_error "Valid providers: ${ALL_PROVIDERS}"
                exit 1
                ;;
        esac
    done
}

main() {
    if [[ $# -eq 0 ]]; then
        usage
        exit 0
    fi

    local providers=()

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --all)
                read -r -a providers <<< "${ALL_PROVIDERS}"
                shift
                ;;
            --help|-h)
                usage
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                providers+=("$1")
                shift
                ;;
        esac
    done

    if [[ ${#providers[@]} -eq 0 ]]; then
        log_error "No providers specified"
        usage
        exit 1
    fi

    download_providers "${providers[@]}"
}

main "$@"
