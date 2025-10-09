# Kubernetes Build Farm Watcher

A configurable Kubernetes watcher that monitors pods and jobs across all namespaces based on label selectors. The watcher automatically restarts at configurable intervals to force cache reloads and supports running multiple concurrent watchers. Optionally includes secrets and jobs listing functionality for API server load testing.

## Features

- **Multi-resource monitoring**: Watches both pods and jobs
- **Label-based filtering**: Each watcher automatically uses a unique label selector (`build-farm-watcher=watcher1`, `build-farm-watcher=watcher2`, etc.)
- **Auto-restart mechanism**: Periodic restarts to force cache reloads
- **Multiple watchers**: Support for running multiple concurrent watcher instances, each watching resources with its specific label
- **Cluster-wide monitoring**: Monitors resources across all namespaces
- **Namespace filtering**: Supports regex-based namespace filtering for listing operations
- **Kubernetes-native**: Designed to run as a Kubernetes deployment
- **Load testing capability**: Optional secrets and jobs listing for API server stress testing

## Configuration

The application is configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RESTART_INTERVAL` | `30m` | How often to restart watchers (Go duration format) |
| `SLEEP_BEFORE_RESTART` | `5s` | Sleep time before restarting watchers (Go duration format) |
| `NUM_WATCHERS` | `3` | Number of concurrent watcher instances (each gets label `build-farm-watcher=watcher<N>`) |
| `LOG_LEVEL` | `info` | Log level (info, debug, warn, error) |
| `ENABLE_POD_WATCHER` | `true` | Enable pod watching via informers |
| `ENABLE_JOB_WATCHER` | `true` | Enable job watching via informers |
| `ENABLE_SECRETS_LISTING` | `false` | Enable periodic secrets listing for load testing |
| `SECRETS_LIST_INTERVAL` | `10s` | How often to list all secrets (Go duration format) |
| `ENABLE_JOBS_LISTING` | `false` | Enable periodic jobs listing for load testing |
| `JOBS_LIST_INTERVAL` | `10s` | How often to list all jobs (Go duration format) |
| `NAMESPACE_FILTER_REGEX` | `""` | Regex pattern to filter namespaces for listing operations (empty = all namespaces) |

## Building

### Local Build
```bash
make build
```

### Container Build
```bash
make container-build
```

## Deployment

### Prerequisites
- Kubernetes cluster access
- kubectl configured

### Deploy to Kubernetes
```bash
# Deploy all resources
make deploy

# Check logs
make logs

# Undeploy
make undeploy
```

### Manual Deployment
```bash
# Deploy all resources (RBAC + application)
kubectl apply -f deploy/k8s-watcher.yaml
```

## Usage Examples

### Basic Setup
Set the number of watchers you want to run. Each watcher will monitor resources with its assigned label:
```yaml
env:
- name: NUM_WATCHERS
  value: "3"
- name: RESTART_INTERVAL
  value: "15m"
```

This creates 3 watchers that will monitor resources with labels:
- `build-farm-watcher=watcher1`
- `build-farm-watcher=watcher2`
- `build-farm-watcher=watcher3`

### Label Your Resources
To have resources monitored by a specific watcher, add the appropriate label to your pods/jobs:
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-build-pod
  labels:
    build-farm-watcher: watcher1
spec:
  # ... pod spec
```

### Load testing with secrets and jobs listing
```yaml
env:
- name: ENABLE_SECRETS_LISTING
  value: "true"
- name: SECRETS_LIST_INTERVAL
  value: "5s"
- name: ENABLE_JOBS_LISTING
  value: "true"
- name: JOBS_LIST_INTERVAL
  value: "5s"
- name: NUM_WATCHERS
  value: "10"
- name: RESTART_INTERVAL
  value: "5m"
```

### Load testing with namespace filtering
```yaml
env:
- name: ENABLE_SECRETS_LISTING
  value: "true"
- name: SECRETS_LIST_INTERVAL
  value: "5s"
- name: ENABLE_JOBS_LISTING
  value: "true"
- name: JOBS_LIST_INTERVAL
  value: "5s"
- name: NAMESPACE_FILTER_REGEX
  value: "^(prod-|staging-).*"
- name: NUM_WATCHERS
  value: "5"
- name: RESTART_INTERVAL
  value: "10m"
```

### Listing only (without watchers)
For pure load testing without the overhead of informer-based watchers:
```yaml
env:
- name: ENABLE_POD_WATCHER
  value: "false"
- name: ENABLE_JOB_WATCHER
  value: "false"
- name: ENABLE_SECRETS_LISTING
  value: "true"
- name: SECRETS_LIST_INTERVAL
  value: "5s"
- name: ENABLE_JOBS_LISTING
  value: "true"
- name: JOBS_LIST_INTERVAL
  value: "5s"
- name: NUM_WATCHERS
  value: "10"
```

## Architecture

The application consists of:

1. **Main Process**: Coordinates multiple watcher goroutines
2. **Watcher Instances**: Each watcher monitors both pods and jobs with its unique label selector (`build-farm-watcher=watcher<N>`)
3. **Restart Mechanism**: Watchers automatically restart at configured intervals
4. **Cache Reload**: Restarts force Kubernetes client cache reloads

### Label Selector Assignment
When you set `NUM_WATCHERS=N`, the application automatically creates N watchers with label selectors:
- Watcher 1: `build-farm-watcher=watcher1`
- Watcher 2: `build-farm-watcher=watcher2`
- ...
- Watcher N: `build-farm-watcher=watcher<N>`

## RBAC Permissions

The application requires the following cluster permissions:
- `get`, `list`, `watch` on `pods` (core API group)
- `get`, `list`, `watch` on `jobs` (batch API group)
- `list` on `secrets` (core API group) - only when secrets listing is enabled
- `list` on `namespaces` (core API group) - only when namespace filtering is enabled

## Monitoring

Check the application logs to see watcher activity:
```bash
kubectl logs -f deployment/k8s-watcher
```

Example log output:
```
Watcher 1: Starting pod watcher with label selector: build-farm-watcher=watcher1
Watcher 1: Pod added: default/build-job-12345, Phase: Running
Watcher 2: Starting job watcher with label selector: build-farm-watcher=watcher2
Watcher 2: Job updated: ci-namespace/compile-job, Conditions: [Complete]
Watcher 1: Listed 1247 secrets across all namespaces (took 89ms)
Watcher 2: Listed 342 jobs across all namespaces (took 45ms)
Watcher 3: Listed 234 secrets across 12 namespaces matching '^(prod-|staging-).*' (took 67ms)
Watcher 3: Listed 89 jobs across 12 namespaces matching '^(prod-|staging-).*' (took 34ms)
```

## Development

### Run Locally
```bash
# Set environment variables
export KUBECONFIG=~/.kube/config
export NUM_WATCHERS=2
export ENABLE_SECRETS_LISTING=false

# Run the application
go run ./cmd/watcher
```

### Testing
```bash
make test
```

### Code Quality
```bash
make fmt vet
```