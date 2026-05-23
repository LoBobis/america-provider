# Unified Controller + GitLab CI Pipeline

## Overview

Consolidate the per-kind controller boilerplate (Topic, DataPower, Service) into a single generic controller that uses the existing `Deployable` interface, and add a GitLab CI pipeline with k3s-based E2E integration tests.

## Part 1: Unified Generic Controller

### Problem

Topic, DataPower, and Service each have their own controller package (`internal/controller/topic/`, `internal/controller/datapower/`) with ~160 lines of nearly identical code. The only differences are:
- Type assertions (`*v1alpha1.Topic` vs `*v1alpha1.DataPower`)
- Error string (`"managed resource is not a Topic custom resource"`)
- GVK/GroupKind metadata used for controller registration
- The `ObjectList` type for state metrics

All CRUD logic already delegates to `common.ConnectExternal`, `common.ObserveExternal`, `common.CreateExternal`, etc. via the `Deployable` interface.

### Solution

Create a single generic controller package at `internal/controller/america/` with:

```go
type ResourceConfig struct {
    Kind             string
    GroupKind        string
    GroupVersionKind schema.GroupVersionKind
    NewObject        func() common.Deployable           // e.g. func() { return &v1alpha1.Topic{} }
    NewObjectList    func() client.ObjectList            // e.g. func() { return &v1alpha1.TopicList{} }
    CastManaged      func(resource.Managed) (common.Deployable, error)
}
```

A single `SetupGated(mgr, opts, webhookEventCh, cfg ResourceConfig)` function registers a controller for any config. The `connector.Connect` calls `cfg.CastManaged(mg)` instead of a hardcoded type assertion. The `external` CRUD methods do the same.

### Registration

In `internal/controller/america.go` (the top-level registration file):

```go
configs := []america.ResourceConfig{
    america.TopicConfig(),
    america.DataPowerConfig(),
    america.ServiceConfig(),
}
for _, cfg := range configs {
    if err := america.SetupGated(mgr, o, webhookEventCh, cfg); err != nil {
        return err
    }
}
```

Each `*Config()` function returns a `ResourceConfig` with the correct types, GVK, and cast function.

### DynamicOperation

Stays as a separate controller in `internal/controller/dynamicoperation/`. It uses a custom non-managed reconciler with predicate filters for fire-and-forget semantics - fundamentally different from the standard Crossplane managed reconciler pattern.

### Files changed

- **New**: `internal/controller/america/controller.go` - generic controller with `ResourceConfig`
- **New**: `internal/controller/america/configs.go` - `TopicConfig()`, `DataPowerConfig()`, `ServiceConfig()`
- **Modified**: `internal/controller/america.go` - loop over configs instead of per-kind imports
- **Deleted**: `internal/controller/topic/topic.go`, `internal/controller/datapower/datapower.go`
- **Kept**: `internal/controller/common/controller_operations.go` (unchanged)
- **Kept**: `internal/controller/dynamicoperation/dynamicoperation.go` (unchanged)

## Part 2: GitLab CI Pipeline

### Stages

1. **test** - Unit tests, vet, lint
2. **build** - Compile Go binary, build Docker image, push to `$PROVIDER_REGISTRY`
3. **integration** - E2E tests on k3s

### Test Stage

- `go test ./...` with race detector
- `go vet ./...`
- Uses `golang:1.24` image

### Build Stage

- Multi-platform Go binary build (`linux/amd64`)
- Docker build using `cluster/images/provider-template/Dockerfile`
- Push to `$PROVIDER_REGISTRY/provider-america:$CI_COMMIT_SHORT_SHA`
- Uses Docker-in-Docker via `docker:dind` service

### Integration Stage

Uses GitLab CI `services` to run k3s (`rancher/k3s:v1.31.4-k3s1`):

1. Wait for k3s to be ready
2. Install CRDs from `package/crds/`
3. Deploy the provider as a Kubernetes Deployment
4. Deploy a mock America API as a Deployment + Service
5. Create Topic, DataPower, and Service custom resources
6. Poll until resources reach Ready/Synced conditions (timeout: 120s)
7. Verify all resources are healthy

The mock America API is a minimal HTTP server that:
- `POST /` returns `{"deployment_id": "<uuid>", "message": "created"}`
- `GET /deployments/<id>` returns `{"id": "<id>", "status": "CREATED", "parameters": {}}`

### CI Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `PROVIDER_REGISTRY` | Container registry URL | (must be set) |
| `PROVIDER_IMAGE_TAG` | Image tag | `$CI_COMMIT_SHORT_SHA` |

### Pipeline file

`.gitlab-ci.yml` at repository root.
