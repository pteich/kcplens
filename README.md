# kcplens

A terminal UI for visualizing and navigating kcp-enabled Kubernetes clusters.

## Overview

`kcplens` provides an intuitive interface for exploring kcp's multi-tenant workspaces, API inheritance (Exports/Bindings), and multi-cluster deployment targets. It makes kcp-specific concepts as easy to navigate as standard Kubernetes resources.

### Features

- **Workspace Explorer**: Navigate kcp's hierarchical workspace structure (root → orgs → teams)
- **API Relationships**: View APIExports and APIBindings with detailed info and YAML inspection
- **Resource Browser**: Discover and list all available resources in a workspace (like `kubectl get widgets`)
- **SyncTarget View**: See attached physical clusters and their status
- **Keyboard-Driven**: Full keyboard navigation with discoverable key bindings

## Installation

Download a pre-compiled binary for your operating system from here: https://github.com/pteich/kcplens/releases
You need just this binary. It works on OSX (Darwin), Linux and Windows.

There are also prebuilt RPM, DEB and APK packages for your Linux distribution.

### Brew

Use Brew to install either with tap:
```shell
brew tap pteich/tap
brew install kcplens
```

or directly:
```shell
brew install pteich/tap/kcplens
```

### Scoop

Use Scoop to install via bucket:
```shell
scoop bucket add pteich https://github.com/pteich/scoop-bucket.git
scoop install kcplens
```
### Go install

You need a local Go installation to use this method.

```shell
go install github.com/pteich/kcplens/cmd/kcplens@latest
```

### Docker

A Docker image is available here: https://github.com/pteich/kcplens/pkgs/container/kcplens
It can be used just like the locally installed binary: 

```shell
docker run ghcr.io/pteich/kcplens:latest -h
```

### Prerequisites
- Go 1.24 or higher

### Build from source
```bash
git clone https://github.com/pteich/kcplens.git
cd kcplens
go build -o kcplens ./cmd/kcplens
mv kcplens /usr/local/bin/
```

## Usage

### Connect to an Existing kcp

```shell
# Use default kubeconfig (~/.kube/config)
./kcplens

# Specify a kubeconfig file
./kcplens -kubeconfig /path/to/kcp/admin.kubeconfig
```

### Key Bindings

| Key | Action |
|-----|--------|
| `a` | View API relationships (APIExports/APIBindings) for current workspace |
| `s` | View SyncTargets (physical clusters) for current workspace |
| `r` | Browse available resources and list instances (like `kubectl get widgets`) |
| `y` | Show YAML of selected API relationship |
| `enter` | Navigate into selected workspace / list selected resource type |
| `backspace` / `esc` | Go back / return to previous view |
| `q` / `ctrl+c` | Quit |

### Navigation

The TUI uses vim-style navigation:
- `j` / `k` or arrow keys to move up/down
- `/` to filter/search the current list
- `enter` to select

You need to first navigate through the available workspaces and press `enter` to select a workspace. Then you can use the other keys to navigate through the available resources and list instances.

## Local Development Environment

A complete kcp test environment can be set up locally using the provided script.

### Quick Start

```shell
./hack/setup-kcp-dev.sh
```

This will:
1. Download kcp v0.30.0
2. Start kcp locally on `127.0.0.1`
3. Create a workspace hierarchy with API providers and consumers
4. Create sample APIExports, APIBindings, and resources

### Workspace Structure Created

```
root
├── org-one
│   ├── team-alpha    (consumes Widget API via APIBinding)
│   └── team-beta     (consumes Widget API via APIBinding)
├── org-two
│   └── team-gamma    (consumes Gadget API via APIBinding)
└── api-provider      (exports Widget & Gadget APIs via APIExport)
```

## Understanding kcp's API Model

kcp separates **API providers** from **API consumers**. This is a key concept that differs from standard Kubernetes.

### The Three Resources

| Resource | Purpose | Lives In |
|----------|---------|----------|
| **APIResourceSchema** | Defines the CRD structure (fields, validation) | API provider workspace |
| **APIExport** | Exposes an API for other workspaces to consume | API provider workspace |
| **APIBinding** | Binds a workspace to an APIExport from another workspace | API consumer workspace |

### Visual Model

```
┌─────────────────────────────────────────────────────────────────┐
│  root:api-provider  (THE API PROVIDER)                          │
│                                                                 │
│  ┌───────────────────────┐    ┌───────────────────────┐         │
│  │ APIResourceSchema     │    │ APIResourceSchema     │         │
│  │ v1.widgets.example.   │    │ v1.gadgets.test.      │         │
│  │ kcp.io                │    │ kcp.io                │         │
│  └───────────┬───────────┘    └───────────┬───────────┘         │
│              │                            │                     │
│  ┌───────────▼───────────┐    ┌───────────▼───────────┐         │
│  │ APIExport             │    │ APIExport             │         │
│  │ example.kcp.io        │    │ test.kcp.io           │         │
│  └───────────┬───────────┘    └───────────────────────┘         │
│              │                                                  │
│              │  I PROVIDE Widget API                            │
└──────────────┼──────────────────────────────────────────────────┘
               │
               │  bound via APIBinding
               │
┌──────────────▼──────────────────────────────────────────────────┐
│  root:org-one:team-alpha  (AN API CONSUMER)                     │
│                                                                 │
│  ┌──────────────────────────────────────────-─┐                 │
│  │ APIBinding: widgets-binding                │                 │
│  │ I CONSUME Widget API from root:api-provider│                 │
│  └───────────────────────┬──────────────────-─┘                 │
│                          │                                      │
│  ┌───────────────────────▼───────────────────────────────────┐  │
│  │ Namespace: demo                                           │  │
│  │                                                           │  │
│  │  ┌─────────────────────┐    ┌─────────────────────┐       │  │
│  │  │ Widget:             │    │ Widget:             │       │  │
│  │  │ widget-alpha-1      │    │ widget-alpha-2      │       │  │
│  │  │ size: large         │    │ size: small         │       │  │
│  │  │ color: blue         │    │ color: red          │       │  │
│  │  └─────────────────────┘    └─────────────────────┘       │  │
│  └───────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Exploring the Setup

After running `./hack/setup-kcp-dev.sh`, try these commands:

```shell
export KUBECONFIG=/path/to/kcplens/.kcp-dev/admin.kubeconfig

# ═══════════════════════════════════════════════════════════════
# THE API PROVIDER WORKSPACE
# ═══════════════════════════════════════════════════════════════
kubectl ws :root:api-provider

# See what APIs are being exported
kubectl get apiexports
# NAME               AGE
# example.kcp.io     5m
# test.kcp.io        5m

# See the underlying schemas (CRD definitions)
kubectl get apiresourceschemas
# NAME                            AGE
# v1.gadgets.test.kcp.io          5m
# v1.widgets.example.kcp.io       5m

# ═══════════════════════════════════════════════════════════════
# THE API CONSUMER WORKSPACE
# ═══════════════════════════════════════════════════════════════
kubectl ws :root:org-one:team-alpha

# See the binding that brings Widget API here
kubectl get apibindings
# NAME                AGE   READY
# widgets-binding     5m    True

# ❌ WRONG: Widgets are NOT in 'default' namespace!
kubectl get widgets
# No resources found in default namespace.

# ✅ CORRECT: Widgets are in 'demo' namespace
kubectl get widgets -n demo
# NAME             SIZE    COLOR   AGE
# widget-alpha-1   large   blue    5m
# widget-alpha-2   small   red     5m

# Or see all widgets across namespaces
kubectl get widgets -A
```

### Common Gotcha

Resources created by the setup script are in the `demo` (or `test`) namespace, not `default`. Always use `-n demo` or `-A`:

```shell
# In team-alpha or team-beta
kubectl get widgets -n demo

# In team-gamma
kubectl get gadgets -n test
```

### Sample APIs

- **example.kcp.io**: Widget CRD with `size`, `color`, `replicas` fields
- **test.kcp.io**: Gadget CRD with `enabled`, `mode` fields

### Script Commands

| Command | Description |
|---------|-------------|
| `./hack/setup-kcp-dev.sh start` | Start KCP and create test resources (default) |
| `./hack/setup-kcp-dev.sh stop` | Stop KCP server |
| `./hack/setup-kcp-dev.sh clean` | Stop KCP and remove all data |
| `./hack/setup-kcp-dev.sh status` | Check if KCP is running |

### Connecting kcplens to Local kcp

```shell
# Start the dev environment
./hack/setup-kcp-dev.sh

# Connect kcplens
./kcplens -kubeconfig .kcp-dev/admin.kubeconfig
```

### Manifests

All kcp resources are stored as YAML files in `hack/manifests/`:

```
hack/manifests/
├── api-provider/           # APIExport and APIResourceSchema definitions
│   ├── apiexport-example.yaml
│   ├── apiexport-test.yaml
│   ├── apiresourceschema-widgets.yaml
│   └── apiresourceschema-gadgets.yaml
├── consumers/              # APIBinding definitions
│   ├── apibinding-widgets.yaml
│   └── apibinding-gadgets.yaml
└── samples/                # Sample resource instances
    ├── widgets-alpha.yaml
    ├── widgets-beta.yaml
    └── gadgets-gamma.yaml
```

## Architecture

```
cmd/kcplens/           # Application entrypoint
internal/
├── kcp/               # kcp client management and discovery
│   ├── client.go      # Client manager, workspace handling
│   └── discovery.go   # Resource discovery, API relationships
└── ui/                # Bubbletea TUI components
    ├── app.go         # Main application state machine
    └── views/         # Individual view components
        ├── workspace_list.go
        ├── api_list.go
        ├── synctarget_list.go
        └── available_resources.go
hack/                  # Development scripts and manifests
├── setup-kcp-dev.sh   # Local kcp environment setup
└── manifests/         # YAML resource definitions

```

## Further Reading

- [kcp Documentation](https://docs.kcp.io/kcp/v0.30/)
- [kcp Workspaces Guide](https://docs.kcp.io/kcp/v0.30/concepts/workspaces/)
- [kcp APIs Guide](https://docs.kcp.io/kcp/v0.30/concepts/apis/)
- [kcp SyncTargets](https://docs.kcp.io/kcp/v0.30/concepts/synctargets/)

## License

MIT
