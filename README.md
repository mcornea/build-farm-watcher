# Kubernetes Build Farm Watcher

A configurable Kubernetes watcher that monitors pods and jobs across all namespaces based on label selectors. The watcher automatically restarts at configurable intervals to force cache reloads and supports running multiple concurrent watchers.

## Features

- **Multi-resource monitoring**: Watches both pods and jobs
- **Label-based filtering**: Configurable label selectors to filter resources
- **Auto-restart mechanism**: Periodic restarts to force cache reloads
- **Multiple watchers**: Support for running multiple concurrent watcher instances
- **Cluster-wide monitoring**: Monitors resources across all namespaces
- **Kubernetes-native**: Designed to run as a Kubernetes deployment

## Configuration

The application is configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `RESTART_INTERVAL` | `30m` | How often to restart watchers (Go duration format) |
| `SLEEP_BEFORE_RESTART` | `5s` | Sleep time before restarting watchers (Go duration format) |
| `NUM_WATCHERS` | `3` | Number of concurrent watcher instances |
| `LABEL_SELECTOR` | `""` | Kubernetes label selector for filtering resources |
| `LOG_LEVEL` | `info` | Log level (info, debug, warn, error) |

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

### Monitor build farm pods
Update the environment variables in the deployment to watch for build farm related resources:
```yaml
env:
- name: LABEL_SELECTOR
  value: "app=build-farm"
- name: NUM_WATCHERS
  value: "5"
- name: RESTART_INTERVAL
  value: "15m"
```

### Monitor CI/CD pipeline jobs
```yaml
env:
- name: LABEL_SELECTOR
  value: "pipeline=ci,component=builder"
- name: NUM_WATCHERS
  value: "2"
- name: RESTART_INTERVAL
  value: "60m"
```

## Architecture

The application consists of:

1. **Main Process**: Coordinates multiple watcher goroutines
2. **Watcher Instances**: Each watcher monitors both pods and jobs
3. **Restart Mechanism**: Watchers automatically restart at configured intervals
4. **Cache Reload**: Restarts force Kubernetes client cache reloads

## RBAC Permissions

The application requires the following cluster permissions:
- `get`, `list`, `watch` on `pods` (core API group)
- `get`, `list`, `watch` on `jobs` (batch API group)

## Monitoring

Check the application logs to see watcher activity:
```bash
kubectl logs -f deployment/k8s-watcher
```

Example log output:
```
Watcher 1: Starting pod watcher with label selector: app=build-farm
Watcher 1: Pod added: default/build-job-12345, Phase: Running
Watcher 2: Job updated: ci-namespace/compile-job, Conditions: [Complete]
```

## Development

### Run Locally
```bash
# Set environment variables
export KUBECONFIG=~/.kube/config
export LABEL_SELECTOR="app=test"
export NUM_WATCHERS=2

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