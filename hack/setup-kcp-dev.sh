#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
MANIFESTS_DIR="${SCRIPT_DIR}/manifests"
KCP_VERSION="${KCP_VERSION:-v0.30.0}"
KCP_DIR="${PROJECT_DIR}/.kcp-dev"
KCP_BIN="${KCP_DIR}/bin"
KUBECONFIG="${KCP_DIR}/admin.kubeconfig"
export KUBECONFIG

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

check_dependencies() {
    log_info "Checking dependencies..."
    local missing=()
    
    command -v kubectl >/dev/null 2>&1 || missing+=("kubectl")
    command -v curl >/dev/null 2>&1 || missing+=("curl")
    command -v tar >/dev/null 2>&1 || missing+=("tar")
    
    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing dependencies: ${missing[*]}"
        exit 1
    fi
    log_info "All dependencies satisfied"
}

detect_os_arch() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *) log_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    log_info "Detected OS: $OS, Arch: $ARCH"
}

download_kcp() {
    if [ -x "${KCP_BIN}/kcp" ]; then
        log_info "KCP binary already exists, skipping download"
    else
        log_info "Downloading KCP ${KCP_VERSION}..."
        mkdir -p "${KCP_BIN}"
        
        local base_url="https://github.com/kcp-dev/kcp/releases/download/${KCP_VERSION}"
        local archive="kcp_${KCP_VERSION//v}_${OS}_${ARCH}.tar.gz"
        local download_url="${base_url}/${archive}"
        
        log_info "Downloading from: ${download_url}"
        curl -sL "${download_url}" | tar -xzf - -C "${KCP_BIN}" --strip-components=1
        
        chmod +x "${KCP_BIN}/kcp"
        log_info "KCP binary installed to ${KCP_BIN}"
    fi
}

install_krew_plugins() {
    log_info "Installing kubectl kcp plugins via krew..."
    
    local krew_path="${KREW_ROOT:-$HOME/.krew}/bin"
    
    if ! kubectl krew version >/dev/null 2>&1; then
        log_warn "krew not installed. Installing krew..."
        (
            set -x; cd "$(mktemp -d)" &&
            OS="$(uname | tr '[:upper:]' '[:lower:]')" &&
            ARCH="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64$/arm64/')" &&
            KREW="krew-${OS}_${ARCH}" &&
            curl -fsSLO "https://github.com/kubernetes-sigs/krew/releases/latest/download/${KREW}.tar.gz" &&
            tar zxvf "${KREW}.tar.gz" &&
            ./${KREW} install krew
        )
    fi
    
    export PATH="${krew_path}:${PATH}"
    
    kubectl krew index add kcp-dev https://github.com/kcp-dev/krew-index.git 2>/dev/null || true
    kubectl krew install kcp-dev/kcp 2>/dev/null || log_warn "kcp plugin may already be installed"
    kubectl krew install kcp-dev/ws 2>/dev/null || log_warn "ws plugin may already be installed"
    kubectl krew install kcp-dev/create-workspace 2>/dev/null || log_warn "create-workspace plugin may already be installed"
    
    log_info "kubectl plugins installed"
}

start_kcp() {
    if [ -f "${KCP_DIR}/kcp.pid" ]; then
        local pid
        pid=$(cat "${KCP_DIR}/kcp.pid")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "KCP already running (PID: $pid)"
            return
        fi
    fi
    
    log_info "Starting KCP server..."
    mkdir -p "${KCP_DIR}"
    
    cd "${KCP_DIR}"
    "${KCP_BIN}/kcp" start \
        --bind-address=127.0.0.1 \
        --kubeconfig-path="${KCP_DIR}/admin.kubeconfig" \
        > "${KCP_DIR}/kcp.log" 2>&1 &
    
    echo $! > "${KCP_DIR}/kcp.pid"
    log_info "KCP started (PID: $(cat "${KCP_DIR}/kcp.pid"))"
}

wait_for_kcp() {
    log_info "Waiting for KCP to be ready..."
    local max_attempts=60
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if kubectl get workspaces.tenancy.kcp.io >/dev/null 2>&1; then
            log_info "KCP is ready!"
            kubectl version --short 2>/dev/null || true
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    
    log_error "KCP failed to become ready after $max_attempts attempts"
    log_error "Check logs at: ${KCP_DIR}/kcp.log"
    exit 1
}

install_kubectl_plugins() {
    export PATH="${KCP_BIN}:${PATH}"
    log_info "kcp binary available at ${KCP_BIN}/kcp"
}

create_workspace_structure() {
    log_info "Creating workspace structure..."
    
    kubectl ws :root
    
    kubectl ws create org-one 2>/dev/null || log_warn "org-one already exists"
    kubectl ws :root:org-one
    kubectl ws create team-alpha 2>/dev/null || log_warn "team-alpha already exists"
    kubectl ws create team-beta 2>/dev/null || log_warn "team-beta already exists"
    
    kubectl ws :root
    kubectl ws create org-two 2>/dev/null || log_warn "org-two already exists"
    kubectl ws :root:org-two
    kubectl ws create team-gamma 2>/dev/null || log_warn "team-gamma already exists"
    
    log_info "Workspace structure created:"
    kubectl ws :root
    kubectl get workspaces -o custom-columns=NAME:.metadata.name,TYPE:.spec.type,PHASE:.status.phase
}

create_api_provider_workspace() {
    log_info "Setting up API provider workspace..."
    
    kubectl ws :root
    kubectl ws create api-provider 2>/dev/null || log_warn "api-provider already exists"
    kubectl ws :root:api-provider
    
    log_info "Applying APIResourceSchema and APIExport for example.kcp.io..."
    kubectl apply -f "${MANIFESTS_DIR}/api-provider/apiresourceschema-widgets.yaml"
    kubectl apply -f "${MANIFESTS_DIR}/api-provider/apiexport-example.yaml"
    
    log_info "API Export created: example.kcp.io"
}

create_api_consumer_workspaces() {
    log_info "Setting up API consumer workspaces..."
    
    kubectl ws :root:org-one:team-alpha
    kubectl apply -f "${MANIFESTS_DIR}/consumers/apibinding-widgets.yaml"
    
    kubectl ws :root:org-one:team-beta
    kubectl apply -f "${MANIFESTS_DIR}/consumers/apibinding-widgets.yaml"
    
    log_info "APIBindings created in consumer workspaces"
}

wait_for_apibinding() {
    local workspace="$1"
    local binding_name="$2"
    local max_attempts=30
    local attempt=0
    
    log_info "Waiting for APIBinding ${binding_name} in ${workspace} to be ready..."
    kubectl ws "${workspace}" >/dev/null 2>&1
    
    while [ $attempt -lt $max_attempts ]; do
        local phase
        phase=$(kubectl get apibinding "${binding_name}" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
        if [ "$phase" = "Bound" ]; then
            log_info "APIBinding ${binding_name} is bound"
            return 0
        fi
        attempt=$((attempt + 1))
        sleep 2
    done
    
    log_warn "APIBinding ${binding_name} not ready after ${max_attempts} attempts, continuing anyway..."
}

create_sample_resources() {
    log_info "Creating sample Widget resources..."
    
    wait_for_apibinding ":root:org-one:team-alpha" "widgets-binding"
    
    kubectl ws :root:org-one:team-alpha
    kubectl create namespace demo 2>/dev/null || true
    kubectl apply -f "${MANIFESTS_DIR}/samples/widgets-alpha.yaml"
    
    wait_for_apibinding ":root:org-one:team-beta" "widgets-binding"
    kubectl ws :root:org-one:team-beta
    kubectl create namespace demo 2>/dev/null || true
    kubectl apply -f "${MANIFESTS_DIR}/samples/widgets-beta.yaml"
    
    log_info "Sample resources created"
}

create_second_api_export() {
    log_info "Creating second API export for testing..."
    
    kubectl ws :root:api-provider
    
    log_info "Applying APIResourceSchema and APIExport for test.kcp.io..."
    kubectl apply -f "${MANIFESTS_DIR}/api-provider/apiresourceschema-gadgets.yaml"
    kubectl apply -f "${MANIFESTS_DIR}/api-provider/apiexport-test.yaml"
    
    kubectl ws :root:org-two:team-gamma
    kubectl apply -f "${MANIFESTS_DIR}/consumers/apibinding-gadgets.yaml"
    
    wait_for_apibinding ":root:org-two:team-gamma" "gadgets-binding"
    kubectl create namespace test 2>/dev/null || true
    kubectl apply -f "${MANIFESTS_DIR}/samples/gadgets-gamma.yaml"
    
    log_info "Second API export and binding created"
}

print_summary() {
    echo ""
    echo "========================================"
    echo -e "${GREEN}KCP Development Environment Ready!${NC}"
    echo "========================================"
    echo ""
    echo "KCP Version: ${KCP_VERSION}"
    echo "Kubeconfig:  ${KUBECONFIG}"
    echo "KCP Log:     ${KCP_DIR}/kcp.log"
    echo ""
    echo "Workspace Structure:"
    echo "  root"
    echo "  ├── org-one"
    echo "  │   ├── team-alpha    (consumes Widget API)"
    echo "  │   └── team-beta     (consumes Widget API)"
    echo "  ├── org-two"
    echo "  │   └── team-gamma    (consumes Gadget API)"
    echo "  └── api-provider      (exports Widget & Gadget APIs)"
    echo ""
    echo "API Model (Provider → Consumer):"
    echo "  api-provider workspace:"
    echo "    - APIResourceSchema: defines Widget/Gadget CRD structure"
    echo "    - APIExport: makes APIs available to other workspaces"
    echo "  team-alpha/beta workspaces:"
    echo "    - APIBinding: connects to api-provider's exports"
    echo "    - Widget resources: actual instances (in 'demo' namespace)"
    echo ""
    echo "Quick Commands:"
    echo "  export KUBECONFIG=${KUBECONFIG}"
    echo ""
    echo "  # Explore the API PROVIDER workspace:"
    echo "  kubectl ws :root:api-provider"
    echo "  kubectl get apiexports              # See exported APIs"
    echo "  kubectl get apiresourceschemas      # See CRD definitions"
    echo ""
    echo "  # Explore an API CONSUMER workspace:"
    echo "  kubectl ws :root:org-one:team-alpha"
    echo "  kubectl get apibindings             # See what APIs are bound"
    echo "  kubectl get widgets -n demo         # Widgets are in 'demo' namespace!"
    echo "  kubectl get widgets -A              # Or see all widgets"
    echo ""
    echo "To stop KCP:"
    echo "  kill \$(cat ${KCP_DIR}/kcp.pid)"
    echo ""
}

stop_kcp() {
    if [ -f "${KCP_DIR}/kcp.pid" ]; then
        local pid
        pid=$(cat "${KCP_DIR}/kcp.pid")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping KCP (PID: $pid)..."
            kill "$pid"
            rm "${KCP_DIR}/kcp.pid"
            log_info "KCP stopped"
        fi
    fi
}

clean() {
    log_info "Cleaning up KCP development environment..."
    stop_kcp
    rm -rf "${KCP_DIR}"
    log_info "Cleanup complete"
}

usage() {
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  start     Start KCP and create test resources (default)"
    echo "  stop      Stop KCP server"
    echo "  clean     Stop KCP and remove all data"
    echo "  status    Check KCP status"
    echo ""
}

status() {
    if [ -f "${KCP_DIR}/kcp.pid" ]; then
        local pid
        pid=$(cat "${KCP_DIR}/kcp.pid")
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "${GREEN}KCP is running${NC} (PID: $pid)"
            echo "Kubeconfig: ${KUBECONFIG}"
            export KUBECONFIG
            kubectl get workspaces -A 2>/dev/null | head -20 || echo "Cannot connect to KCP"
        else
            echo -e "${RED}KCP PID file exists but process is not running${NC}"
        fi
    else
        echo -e "${YELLOW}KCP is not running${NC}"
    fi
}

main() {
    local command="${1:-start}"
    
    case "$command" in
        start)
            check_dependencies
            detect_os_arch
            download_kcp
            install_krew_plugins
            start_kcp
            wait_for_kcp
            install_kubectl_plugins
            create_workspace_structure
            create_api_provider_workspace
            create_api_consumer_workspaces
            create_sample_resources
            create_second_api_export
            print_summary
            ;;
        stop)
            stop_kcp
            ;;
        clean)
            clean
            ;;
        status)
            status
            ;;
        -h|--help)
            usage
            ;;
        *)
            log_error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

main "$@"
