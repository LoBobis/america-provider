# Unified Controller + GitLab CI Pipeline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Consolidate per-kind controller boilerplate for Topic/DataPower/Service into one generic controller, and add a GitLab CI pipeline with k3s E2E integration tests.

**Architecture:** A single generic controller uses a `ResourceConfig` struct to register per-kind controllers with the Crossplane managed reconciler. DynamicOperation stays separate with its custom reconciler. GitLab CI uses Docker-in-Docker for image build and a k3s service container for E2E tests with a mock America API.

**Tech Stack:** Go 1.24, Crossplane Runtime v2, controller-runtime, GitLab CI, k3s, Docker

---

## File Structure

### New files
- `internal/controller/america/controller.go` — Generic controller with `ResourceConfig`, `Setup`, `SetupGated`, `connector`, `external`
- `internal/controller/america/configs.go` — `TopicConfig()`, `DataPowerConfig()`, `ServiceConfig()` factory functions
- `internal/controller/america/controller_test.go` — Unit tests for the generic controller
- `.gitlab-ci.yml` — GitLab CI pipeline
- `test/e2e/mock-america/main.go` — Mock America API server for E2E tests
- `test/e2e/mock-america/Dockerfile` — Dockerfile for mock server
- `test/e2e/manifests/provider-deployment.yaml` — Provider k8s deployment manifest
- `test/e2e/manifests/mock-america.yaml` — Mock America API k8s manifests
- `test/e2e/manifests/test-resources.yaml` — Test Topic/DataPower/Service CRs
- `test/e2e/run-e2e.sh` — E2E test script

### Modified files
- `internal/controller/america.go` — Replace per-kind imports with loop over `ResourceConfig`s

### Deleted files
- `internal/controller/topic/topic.go` — Replaced by generic controller
- `internal/controller/topic/topic_test.go` — Replaced by generic controller test
- `internal/controller/datapower/datapower.go` — Replaced by generic controller

### Unchanged files
- `internal/controller/common/controller_operations.go` — Shared CRUD logic (unchanged)
- `internal/controller/dynamicoperation/dynamicoperation.go` — Separate controller (unchanged)
- `internal/controller/config/config.go` — ProviderConfig controller (unchanged)
- `apis/middleware/v1alpha1/*.go` — API types (unchanged)

---

## Task 1: Copy source files from romantic-robinson worktree

Before building the unified controller, we need the API types, client, common controller, DynamicOperation controller, webhook, main.go, and supporting files from the romantic-robinson worktree into this worktree. This worktree currently only has the template (MyType) files.

**Files:**
- Copy from: `C:\Users\eyalk\OneDrive\Documents\provider-template\.claude\worktrees\romantic-robinson\`
- Copy to: current worktree

- [ ] **Step 1: Copy API types and middleware package**

Copy the entire `apis/` directory from romantic-robinson, replacing the template's `apis/`:

```bash
# Remove template sample API
rm -rf apis/sample

# Copy middleware API types
cp -r "../romantic-robinson/apis/middleware" apis/middleware

# Copy updated top-level apis files
cp "../romantic-robinson/apis/america.go" apis/america.go

# Copy updated v1alpha1 types (ProviderConfig with AmericaConfig)
cp "../romantic-robinson/apis/v1alpha1/types.go" apis/v1alpha1/types.go
cp "../romantic-robinson/apis/v1alpha1/register.go" apis/v1alpha1/register.go
cp "../romantic-robinson/apis/v1alpha1/doc.go" apis/v1alpha1/doc.go
cp "../romantic-robinson/apis/v1alpha1/zz_generated.deepcopy.go" apis/v1alpha1/zz_generated.deepcopy.go
cp "../romantic-robinson/apis/v1alpha1/zz_generated.pc.go" apis/v1alpha1/zz_generated.pc.go
cp "../romantic-robinson/apis/v1alpha1/zz_generated.pcu.go" apis/v1alpha1/zz_generated.pcu.go
cp "../romantic-robinson/apis/v1alpha1/zz_generated.pculist.go" apis/v1alpha1/zz_generated.pculist.go
```

- [ ] **Step 2: Copy internal packages**

```bash
# Copy America client
cp -r "../romantic-robinson/internal/clients" internal/clients

# Copy common controller operations
cp -r "../romantic-robinson/internal/controller/common" internal/controller/common

# Copy DynamicOperation controller
cp -r "../romantic-robinson/internal/controller/dynamicoperation" internal/controller/dynamicoperation

# Copy webhook server
cp -r "../romantic-robinson/internal/webhook" internal/webhook

# Copy version package
cp "../romantic-robinson/internal/version/version.go" internal/version/version.go

# Copy config controller
cp "../romantic-robinson/internal/controller/config/config.go" internal/controller/config/config.go
```

- [ ] **Step 3: Copy main.go and build files**

```bash
cp "../romantic-robinson/cmd/provider/main.go" cmd/provider/main.go
cp "../romantic-robinson/go.mod" go.mod
cp "../romantic-robinson/go.sum" go.sum
cp "../romantic-robinson/Makefile" Makefile
cp "../romantic-robinson/cluster/images/provider-template/Dockerfile" cluster/images/provider-template/Dockerfile
cp "../romantic-robinson/package/crossplane.yaml" package/crossplane.yaml
```

- [ ] **Step 4: Copy CRDs and examples**

```bash
cp -r "../romantic-robinson/package/crds" package/crds
cp -r "../romantic-robinson/examples" examples
```

- [ ] **Step 5: Remove old template files that are no longer needed**

```bash
# Remove old template controller and sample types
rm -rf internal/controller/mytype
rm -f internal/controller/register.go
rm -f apis/template.go
rm -rf apis/sample
```

- [ ] **Step 6: Commit the source file migration**

```bash
git add -A
git commit -m "feat: Copy America provider source files from romantic-robinson worktree"
```

---

## Task 2: Create the generic controller

**Files:**
- Create: `internal/controller/america/controller.go`

- [ ] **Step 1: Create the `internal/controller/america/` directory**

```bash
mkdir -p internal/controller/america
```

- [ ] **Step 2: Write the generic controller**

Create `internal/controller/america/controller.go`:

```go
package america

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	xpevent "github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	apisv1alpha1 "github.com/crossplane/provider-america/apis/v1alpha1"
	americaClient "github.com/crossplane/provider-america/internal/clients/america"
	"github.com/crossplane/provider-america/internal/controller/common"
)

// ResourceConfig holds all type-specific values needed to register a controller
// for a Deployable managed resource.
type ResourceConfig struct {
	Kind             string
	GroupKind        string
	GroupVersionKind schema.GroupVersionKind
	NewObject        func() common.Deployable
	NewObjectList    func() client.ObjectList
	CastManaged      func(resource.Managed) (common.Deployable, error)
}

// SetupGated registers a gated controller for the given resource config.
func SetupGated(mgr ctrl.Manager, o controller.Options, webhookEventCh chan event.GenericEvent, cfg ResourceConfig) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o, webhookEventCh, cfg); err != nil {
			panic(errors.Wrapf(err, "cannot setup %s controller", cfg.Kind))
		}
	}, cfg.GroupVersionKind)
	return nil
}

// Setup registers a managed reconciler controller for the given resource config.
func Setup(mgr ctrl.Manager, o controller.Options, webhookEventCh chan event.GenericEvent, cfg ResourceConfig) error {
	name := managed.ControllerName(cfg.GroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			logger:             o.Logger,
			kube:               mgr.GetClient(),
			usage:              resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newAmericaClientFn: americaClient.NewClient,
			cfg:                cfg,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(xpevent.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
	}

	if o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	if o.Features.Enabled(feature.EnableAlphaChangeLogs) {
		opts = append(opts, managed.WithChangeLogger(o.ChangeLogOptions.ChangeLogger))
	}

	if o.MetricOptions != nil {
		opts = append(opts, managed.WithMetricRecorder(o.MetricOptions.MRMetrics))
	}

	if o.MetricOptions != nil && o.MetricOptions.MRStateMetrics != nil {
		stateMetricsRecorder := statemetrics.NewMRStateRecorder(
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, cfg.NewObjectList(), o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrapf(err, "cannot register MR state metrics recorder for kind %s", cfg.Kind)
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(cfg.GroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(cfg.NewObject().(client.Object)).
		WatchesRawSource(source.Channel(webhookEventCh, &handler.EnqueueRequestForObject{})).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	logger             logging.Logger
	kube               client.Client
	usage              *resource.ProviderConfigUsageTracker
	newAmericaClientFn func(log logging.Logger, creds string) (americaClient.Client, error)
	cfg                ResourceConfig
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, err := c.cfg.CastManaged(mg)
	if err != nil {
		return nil, err
	}

	l, americaConfig, kube, america_client, err := common.ConnectExternal(
		c.logger, c.kube, c.usage, c.newAmericaClientFn, ctx, mg, cr,
	)
	if err != nil {
		return nil, err
	}

	return &external{
		logger:         l,
		americaConfig:  americaConfig,
		kube:           kube,
		america_client: america_client,
		cfg:            c.cfg,
	}, nil
}

type external struct {
	logger         logging.Logger
	americaConfig  apisv1alpha1.AmericaConfig
	kube           client.Client
	america_client americaClient.Client
	cfg            ResourceConfig
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, err := e.cfg.CastManaged(mg)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	errNotFound := fmt.Sprintf("managed resource is not a %s custom resource", e.cfg.Kind)
	return common.ObserveExternal(ctx, e.america_client, e.americaConfig, e.kube, cr, errNotFound)
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, err := e.cfg.CastManaged(mg)
	if err != nil {
		return managed.ExternalCreation{}, err
	}
	return common.CreateExternal(ctx, e.america_client, e.americaConfig, e.kube, cr)
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, err := e.cfg.CastManaged(mg)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}
	return common.UpdateExternal(ctx, e.america_client, e.americaConfig, e.kube, cr)
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, err := e.cfg.CastManaged(mg)
	if err != nil {
		return managed.ExternalDelete{}, err
	}
	return common.DeleteExternal(e.america_client, e.americaConfig, cr)
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}
```

- [ ] **Step 3: Verify the file compiles (will fail until configs.go exists)**

```bash
cd internal/controller/america && go vet ./... 2>&1 || echo "Expected: may fail until configs.go is added"
```

- [ ] **Step 4: Commit**

```bash
git add internal/controller/america/controller.go
git commit -m "feat: Add generic controller for Deployable resources"
```

---

## Task 3: Create resource config factories

**Files:**
- Create: `internal/controller/america/configs.go`

- [ ] **Step 1: Write the config factory functions**

Create `internal/controller/america/configs.go`:

```go
package america

import (
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/crossplane/provider-america/apis/middleware/v1alpha1"
	"github.com/crossplane/provider-america/internal/controller/common"
)

func TopicConfig() ResourceConfig {
	return ResourceConfig{
		Kind:             v1alpha1.TopicKind,
		GroupKind:        v1alpha1.TopicGroupKind,
		GroupVersionKind: v1alpha1.TopicGroupVersionKind,
		NewObject:        func() common.Deployable { return &v1alpha1.Topic{} },
		NewObjectList:    func() client.ObjectList { return &v1alpha1.TopicList{} },
		CastManaged: func(mg resource.Managed) (common.Deployable, error) {
			cr, ok := mg.(*v1alpha1.Topic)
			if !ok {
				return nil, fmt.Errorf("managed resource is not a %s custom resource", v1alpha1.TopicKind)
			}
			return cr, nil
		},
	}
}

func DataPowerConfig() ResourceConfig {
	return ResourceConfig{
		Kind:             v1alpha1.DataPowerKind,
		GroupKind:        v1alpha1.DataPowerGroupKind,
		GroupVersionKind: v1alpha1.DataPowerGroupVersionKind,
		NewObject:        func() common.Deployable { return &v1alpha1.DataPower{} },
		NewObjectList:    func() client.ObjectList { return &v1alpha1.DataPowerList{} },
		CastManaged: func(mg resource.Managed) (common.Deployable, error) {
			cr, ok := mg.(*v1alpha1.DataPower)
			if !ok {
				return nil, fmt.Errorf("managed resource is not a %s custom resource", v1alpha1.DataPowerKind)
			}
			return cr, nil
		},
	}
}

func ServiceConfig() ResourceConfig {
	return ResourceConfig{
		Kind:             v1alpha1.ServiceKind,
		GroupKind:        v1alpha1.ServiceGroupKind,
		GroupVersionKind: v1alpha1.ServiceGroupVersionKind,
		NewObject:        func() common.Deployable { return &v1alpha1.Service{} },
		NewObjectList:    func() client.ObjectList { return &v1alpha1.ServiceList{} },
		CastManaged: func(mg resource.Managed) (common.Deployable, error) {
			cr, ok := mg.(*v1alpha1.Service)
			if !ok {
				return nil, fmt.Errorf("managed resource is not a %s custom resource", v1alpha1.ServiceKind)
			}
			return cr, nil
		},
	}
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go vet ./internal/controller/america/...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/controller/america/configs.go
git commit -m "feat: Add TopicConfig, DataPowerConfig, ServiceConfig factories"
```

---

## Task 4: Update top-level registration and remove old per-kind controllers

**Files:**
- Modify: `internal/controller/america.go`
- Delete: `internal/controller/topic/topic.go`
- Delete: `internal/controller/topic/topic_test.go`
- Delete: `internal/controller/datapower/datapower.go`

- [ ] **Step 1: Rewrite `internal/controller/america.go`**

Replace the entire file with:

```go
package controller

import (
	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/crossplane/provider-america/internal/controller/america"
	"github.com/crossplane/provider-america/internal/controller/config"
	"github.com/crossplane/provider-america/internal/controller/dynamicoperation"
)

// SetupGated creates all America controllers with safe-start support and adds them to
// the supplied manager.
func SetupGated(mgr ctrl.Manager, o controller.Options, webhookEventCh chan event.GenericEvent) error {
	if err := config.Setup(mgr, o, webhookEventCh); err != nil {
		return err
	}

	for _, cfg := range []america.ResourceConfig{
		america.TopicConfig(),
		america.DataPowerConfig(),
		america.ServiceConfig(),
	} {
		if err := america.SetupGated(mgr, o, webhookEventCh, cfg); err != nil {
			return err
		}
	}

	if err := dynamicoperation.Setup(mgr, o.Logger); err != nil {
		return err
	}

	return nil
}
```

- [ ] **Step 2: Delete old per-kind controller packages**

```bash
rm -rf internal/controller/topic
rm -rf internal/controller/datapower
```

- [ ] **Step 3: Verify the full project compiles**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 4: Run existing tests**

```bash
go test ./...
```

Expected: all pass (the deleted topic_test.go only had empty TODO cases)

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor: Replace per-kind controllers with unified generic controller"
```

---

## Task 5: Add unit tests for the generic controller

**Files:**
- Create: `internal/controller/america/controller_test.go`

- [ ] **Step 1: Write unit tests**

Create `internal/controller/america/controller_test.go`:

```go
package america

import (
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/test"
	"github.com/google/go-cmp/cmp"

	v1alpha1 "github.com/crossplane/provider-america/apis/middleware/v1alpha1"
	"github.com/crossplane/provider-america/internal/controller/common"
)

func TestTopicConfig(t *testing.T) {
	cfg := TopicConfig()

	if cfg.Kind != "Topic" {
		t.Errorf("TopicConfig().Kind = %q, want %q", cfg.Kind, "Topic")
	}
	if cfg.GroupVersionKind != v1alpha1.TopicGroupVersionKind {
		t.Errorf("TopicConfig().GroupVersionKind = %v, want %v", cfg.GroupVersionKind, v1alpha1.TopicGroupVersionKind)
	}

	obj := cfg.NewObject()
	if _, ok := obj.(*v1alpha1.Topic); !ok {
		t.Errorf("TopicConfig().NewObject() returned %T, want *v1alpha1.Topic", obj)
	}

	list := cfg.NewObjectList()
	if _, ok := list.(*v1alpha1.TopicList); !ok {
		t.Errorf("TopicConfig().NewObjectList() returned %T, want *v1alpha1.TopicList", list)
	}
}

func TestDataPowerConfig(t *testing.T) {
	cfg := DataPowerConfig()

	if cfg.Kind != "DataPower" {
		t.Errorf("DataPowerConfig().Kind = %q, want %q", cfg.Kind, "DataPower")
	}
	if cfg.GroupVersionKind != v1alpha1.DataPowerGroupVersionKind {
		t.Errorf("DataPowerConfig().GroupVersionKind = %v, want %v", cfg.GroupVersionKind, v1alpha1.DataPowerGroupVersionKind)
	}

	obj := cfg.NewObject()
	if _, ok := obj.(*v1alpha1.DataPower); !ok {
		t.Errorf("DataPowerConfig().NewObject() returned %T, want *v1alpha1.DataPower", obj)
	}
}

func TestServiceConfig(t *testing.T) {
	cfg := ServiceConfig()

	if cfg.Kind != "Service" {
		t.Errorf("ServiceConfig().Kind = %q, want %q", cfg.Kind, "Service")
	}
	if cfg.GroupVersionKind != v1alpha1.ServiceGroupVersionKind {
		t.Errorf("ServiceConfig().GroupVersionKind = %v, want %v", cfg.GroupVersionKind, v1alpha1.ServiceGroupVersionKind)
	}

	obj := cfg.NewObject()
	if _, ok := obj.(*v1alpha1.Service); !ok {
		t.Errorf("ServiceConfig().NewObject() returned %T, want *v1alpha1.Service", obj)
	}
}

func TestCastManaged_Success(t *testing.T) {
	cases := map[string]struct {
		cfg ResourceConfig
		mg  resource.Managed
	}{
		"Topic": {
			cfg: TopicConfig(),
			mg:  &v1alpha1.Topic{},
		},
		"DataPower": {
			cfg: DataPowerConfig(),
			mg:  &v1alpha1.DataPower{},
		},
		"Service": {
			cfg: ServiceConfig(),
			mg:  &v1alpha1.Service{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := tc.cfg.CastManaged(tc.mg)
			if err != nil {
				t.Fatalf("CastManaged() unexpected error: %v", err)
			}
			if got == nil {
				t.Fatal("CastManaged() returned nil")
			}
		})
	}
}

func TestCastManaged_WrongType(t *testing.T) {
	cases := map[string]struct {
		cfg      ResourceConfig
		mg       resource.Managed
		wantKind string
	}{
		"TopicGivenDataPower": {
			cfg:      TopicConfig(),
			mg:       &v1alpha1.DataPower{},
			wantKind: "Topic",
		},
		"DataPowerGivenTopic": {
			cfg:      DataPowerConfig(),
			mg:       &v1alpha1.Topic{},
			wantKind: "DataPower",
		},
		"ServiceGivenTopic": {
			cfg:      ServiceConfig(),
			mg:       &v1alpha1.Topic{},
			wantKind: "Service",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := tc.cfg.CastManaged(tc.mg)
			if err == nil {
				t.Fatal("CastManaged() expected error, got nil")
			}
			wantErr := fmt.Sprintf("managed resource is not a %s custom resource", tc.wantKind)
			if diff := cmp.Diff(wantErr, err.Error(), test.EquateErrors()); diff != "" {
				t.Errorf("CastManaged() error: -want, +got:\n%s", diff)
			}
		})
	}
}

// Compile-time check: all configs produce a common.Deployable.
var _ common.Deployable = TopicConfig().NewObject()
var _ common.Deployable = DataPowerConfig().NewObject()
var _ common.Deployable = ServiceConfig().NewObject()
```

- [ ] **Step 2: Run the tests**

```bash
go test ./internal/controller/america/... -v
```

Expected: all pass

- [ ] **Step 3: Commit**

```bash
git add internal/controller/america/controller_test.go
git commit -m "test: Add unit tests for generic controller configs"
```

---

## Task 6: Create the mock America API server

**Files:**
- Create: `test/e2e/mock-america/main.go`
- Create: `test/e2e/mock-america/Dockerfile`

- [ ] **Step 1: Create the mock server**

Create `test/e2e/mock-america/main.go`:

```go
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type deployment struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Parameters map[string]interface{} `json:"parameters"`
}

var (
	deployments = map[string]*deployment{}
	mu          sync.Mutex
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/" {
			handleCreate(w, r)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/deployments/") {
			handleGet(w, r)
			return
		}
		if r.Method == http.MethodPost && r.URL.Path == "/operations" {
			handleCreateOperation(w, r)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/operations/") {
			handleGetOperation(w, r)
			return
		}
		http.Error(w, "not found", http.StatusNotFound)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Println("Mock America API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ResourceType string `json:"resource_type"`
		Name         string `json:"name"`
		Parameters   string `json:"parameters"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id := uuid.New().String()
	mu.Lock()
	deployments[id] = &deployment{
		ID:     id,
		Name:   req.Name,
		Status: "CREATED",
	}
	mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"deployment_id": id,
		"message":       "created",
	})
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/deployments/")

	mu.Lock()
	dep, ok := deployments[id]
	mu.Unlock()

	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dep)
}

func handleCreateOperation(w http.ResponseWriter, r *http.Request) {
	id := uuid.New().String()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"operation_id": id,
		"status":       "COMPLETED",
	})
}

func handleGetOperation(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/operations/")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"operation_id": id,
		"status":       "COMPLETED",
		"result":       map[string]string{"message": "done"},
	})
}

func init() {
	fmt.Println("Mock America API server starting...")
}
```

- [ ] **Step 2: Create the Dockerfile for mock server**

Create `test/e2e/mock-america/Dockerfile`:

```dockerfile
FROM golang:1.24 AS builder
WORKDIR /app
COPY main.go .
RUN go mod init mock-america && go mod tidy && CGO_ENABLED=0 go build -o mock-america .

FROM gcr.io/distroless/static
COPY --from=builder /app/mock-america /usr/local/bin/mock-america
ENTRYPOINT ["mock-america"]
```

- [ ] **Step 3: Commit**

```bash
git add test/e2e/mock-america/
git commit -m "feat: Add mock America API server for E2E tests"
```

---

## Task 7: Create E2E test manifests

**Files:**
- Create: `test/e2e/manifests/provider-deployment.yaml`
- Create: `test/e2e/manifests/mock-america.yaml`
- Create: `test/e2e/manifests/test-resources.yaml`
- Create: `test/e2e/manifests/provider-config.yaml`

- [ ] **Step 1: Create the provider deployment manifest**

Create `test/e2e/manifests/provider-deployment.yaml`:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: crossplane-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: provider-america
  namespace: crossplane-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: provider-america
  template:
    metadata:
      labels:
        app: provider-america
    spec:
      serviceAccountName: provider-america
      containers:
        - name: provider
          image: ${PROVIDER_IMAGE}
          args:
            - --debug
          ports:
            - containerPort: 8081
              name: webhook
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: provider-america
  namespace: crossplane-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: provider-america
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
  - kind: ServiceAccount
    name: provider-america
    namespace: crossplane-system
```

- [ ] **Step 2: Create mock America API manifest**

Create `test/e2e/manifests/mock-america.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: mock-america
  namespace: crossplane-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mock-america
  template:
    metadata:
      labels:
        app: mock-america
    spec:
      containers:
        - name: mock-america
          image: ${MOCK_AMERICA_IMAGE}
          ports:
            - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: mock-america
  namespace: crossplane-system
spec:
  selector:
    app: mock-america
  ports:
    - port: 8080
      targetPort: 8080
```

- [ ] **Step 3: Create ProviderConfig manifest**

Create `test/e2e/manifests/provider-config.yaml`:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: america-creds
  namespace: default
type: Opaque
stringData:
  credentials: "test-token"
---
apiVersion: america.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
  namespace: default
spec:
  credentials:
    source: Secret
    secretRef:
      name: america-creds
      namespace: default
      key: credentials
  americaConfig:
    americaURL: "http://mock-america.crossplane-system.svc.cluster.local:8080"
    jwtKey: "test"
    projectID: "test-project"
```

- [ ] **Step 4: Create test resources manifest**

Create `test/e2e/manifests/test-resources.yaml`:

```yaml
apiVersion: middleware.america.crossplane.io/v1alpha1
kind: Topic
metadata:
  name: test-topic
  namespace: default
spec:
  providerConfigRef:
    name: default
  region: "us-east-1"
  environment: "dev"
  bundle_id: "test-bundle"
  forProvider:
    resource_type: "KafkaAdvancedConfig1_3"
    properties:
      name: "test-topic"
      description: "E2E test topic"
      max_message_size: 1048576
      max_consumer_groups: 10
      retention_time: 86400
      num_of_partitions: 3
    resource_info: null
---
apiVersion: middleware.america.crossplane.io/v1alpha1
kind: DataPower
metadata:
  name: test-datapower
  namespace: default
spec:
  providerConfigRef:
    name: default
  region: "us-east-1"
  environment: "dev"
  bundle_id: "test-bundle"
  forProvider:
    resource_type: "DataPowerAdvanced1_0"
    properties:
      name: "test-datapower"
      port: 443
      memory: 2048
      enable_t_l_s: true
    resource_info: null
---
apiVersion: middleware.america.crossplane.io/v1alpha1
kind: Service
metadata:
  name: test-service
  namespace: default
spec:
  providerConfigRef:
    name: default
  bundle_id: "test-bundle"
  forProvider:
    serviceName: "test-service"
    servicePassword: "test-pass"
```

- [ ] **Step 5: Commit**

```bash
git add test/e2e/manifests/
git commit -m "feat: Add E2E test Kubernetes manifests"
```

---

## Task 8: Create the E2E test script

**Files:**
- Create: `test/e2e/run-e2e.sh`

- [ ] **Step 1: Write the E2E test runner script**

Create `test/e2e/run-e2e.sh`:

```bash
#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUBECONFIG="${KUBECONFIG:-/etc/rancher/k3s/k3s.yaml}"
TIMEOUT="${E2E_TIMEOUT:-180}"
PROVIDER_IMAGE="${PROVIDER_IMAGE:?PROVIDER_IMAGE must be set}"
MOCK_AMERICA_IMAGE="${MOCK_AMERICA_IMAGE:?MOCK_AMERICA_IMAGE must be set}"

export KUBECONFIG

echo "=== Waiting for k3s to be ready ==="
until kubectl get nodes 2>/dev/null | grep -q " Ready"; do
  echo "Waiting for k3s node..."
  sleep 5
done
echo "k3s is ready"

echo "=== Installing CRDs ==="
kubectl apply -R -f "${SCRIPT_DIR}/../../package/crds/"
kubectl wait --for=condition=Established --all crd --timeout=60s

echo "=== Deploying mock America API ==="
sed "s|\${MOCK_AMERICA_IMAGE}|${MOCK_AMERICA_IMAGE}|g" \
  "${SCRIPT_DIR}/manifests/mock-america.yaml" | kubectl apply -f -

echo "=== Deploying provider ==="
sed "s|\${PROVIDER_IMAGE}|${PROVIDER_IMAGE}|g" \
  "${SCRIPT_DIR}/manifests/provider-deployment.yaml" | kubectl apply -f -

echo "=== Waiting for mock America API ==="
kubectl -n crossplane-system wait --for=condition=Available deployment/mock-america --timeout=120s

echo "=== Waiting for provider ==="
kubectl -n crossplane-system wait --for=condition=Available deployment/provider-america --timeout=120s

echo "=== Creating ProviderConfig ==="
kubectl apply -f "${SCRIPT_DIR}/manifests/provider-config.yaml"
sleep 5

echo "=== Creating test resources ==="
kubectl apply -f "${SCRIPT_DIR}/manifests/test-resources.yaml"

echo "=== Waiting for resources to become Ready ==="
ELAPSED=0
INTERVAL=10
ALL_READY=false

while [ "$ELAPSED" -lt "$TIMEOUT" ]; do
  READY_COUNT=0
  TOTAL=3

  for resource in "topic/test-topic" "datapower/test-datapower" "service/test-service"; do
    KIND=$(echo "$resource" | cut -d/ -f1)
    NAME=$(echo "$resource" | cut -d/ -f2)
    STATUS=$(kubectl get "$KIND" "$NAME" -n default -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
    if [ "$STATUS" = "True" ]; then
      READY_COUNT=$((READY_COUNT + 1))
      echo "  $KIND/$NAME: Ready"
    else
      echo "  $KIND/$NAME: Not Ready (status=$STATUS)"
    fi
  done

  if [ "$READY_COUNT" -eq "$TOTAL" ]; then
    ALL_READY=true
    break
  fi

  echo "  ($READY_COUNT/$TOTAL ready, elapsed ${ELAPSED}s)"
  sleep "$INTERVAL"
  ELAPSED=$((ELAPSED + INTERVAL))
done

echo ""
echo "=== Final resource status ==="
kubectl get topic,datapower,service -n default -o wide 2>/dev/null || true

if [ "$ALL_READY" = true ]; then
  echo ""
  echo "=== E2E TESTS PASSED ==="
  exit 0
else
  echo ""
  echo "=== E2E TESTS FAILED ==="
  echo "Not all resources reached Ready state within ${TIMEOUT}s"
  echo ""
  echo "=== Provider logs ==="
  kubectl -n crossplane-system logs deployment/provider-america --tail=50 2>/dev/null || true
  exit 1
fi
```

- [ ] **Step 2: Make it executable**

```bash
chmod +x test/e2e/run-e2e.sh
```

- [ ] **Step 3: Commit**

```bash
git add test/e2e/run-e2e.sh
git commit -m "feat: Add E2E test runner script"
```

---

## Task 9: Create the GitLab CI pipeline

**Files:**
- Create: `.gitlab-ci.yml`

- [ ] **Step 1: Write the GitLab CI configuration**

Create `.gitlab-ci.yml`:

```yaml
stages:
  - test
  - build
  - integration

variables:
  GO_VERSION: "1.24"
  PROVIDER_IMAGE_TAG: ${CI_COMMIT_SHORT_SHA}

# ──────────────────────────────────────────────
# Stage: test
# ──────────────────────────────────────────────

unit-tests:
  stage: test
  image: golang:${GO_VERSION}
  script:
    - go vet ./...
    - go test -race -count=1 ./...
  cache:
    key: go-mod-${CI_COMMIT_REF_SLUG}
    paths:
      - .go/pkg/mod/
  variables:
    GOPATH: ${CI_PROJECT_DIR}/.go

# ──────────────────────────────────────────────
# Stage: build
# ──────────────────────────────────────────────

build-provider:
  stage: build
  image: docker:27
  services:
    - docker:27-dind
  variables:
    DOCKER_TLS_CERTDIR: "/certs"
  before_script:
    - docker login -u "${REGISTRY_USER}" -p "${REGISTRY_PASSWORD}" "${PROVIDER_REGISTRY}"
  script:
    # Build Go binary
    - docker run --rm
        -v "${CI_PROJECT_DIR}:/src"
        -w /src
        golang:${GO_VERSION}
        sh -c "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-X github.com/crossplane/provider-america/internal/version.Version=${PROVIDER_IMAGE_TAG}' -o cluster/images/provider-template/bin/linux_amd64/provider ./cmd/provider"
    # Build and push provider image
    - docker build
        --build-arg TARGETOS=linux
        --build-arg TARGETARCH=amd64
        -t "${PROVIDER_REGISTRY}/provider-america:${PROVIDER_IMAGE_TAG}"
        cluster/images/provider-template/
    - docker push "${PROVIDER_REGISTRY}/provider-america:${PROVIDER_IMAGE_TAG}"
    # Build and push mock America API image
    - docker build
        -t "${PROVIDER_REGISTRY}/mock-america:${PROVIDER_IMAGE_TAG}"
        test/e2e/mock-america/
    - docker push "${PROVIDER_REGISTRY}/mock-america:${PROVIDER_IMAGE_TAG}"

build-provider-arm64:
  stage: build
  image: docker:27
  services:
    - docker:27-dind
  variables:
    DOCKER_TLS_CERTDIR: "/certs"
  before_script:
    - docker login -u "${REGISTRY_USER}" -p "${REGISTRY_PASSWORD}" "${PROVIDER_REGISTRY}"
  script:
    - docker run --rm
        -v "${CI_PROJECT_DIR}:/src"
        -w /src
        golang:${GO_VERSION}
        sh -c "CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags '-X github.com/crossplane/provider-america/internal/version.Version=${PROVIDER_IMAGE_TAG}' -o cluster/images/provider-template/bin/linux_arm64/provider ./cmd/provider"
    - docker build
        --build-arg TARGETOS=linux
        --build-arg TARGETARCH=arm64
        -t "${PROVIDER_REGISTRY}/provider-america:${PROVIDER_IMAGE_TAG}-arm64"
        cluster/images/provider-template/
    - docker push "${PROVIDER_REGISTRY}/provider-america:${PROVIDER_IMAGE_TAG}-arm64"
  rules:
    - when: manual
      allow_failure: true

# ──────────────────────────────────────────────
# Stage: integration
# ──────────────────────────────────────────────

e2e-tests:
  stage: integration
  image: alpine/k8s:1.31.4
  services:
    - name: rancher/k3s:v1.31.4-k3s1
      alias: k3s
      command: ["server", "--disable=traefik", "--disable=metrics-server", "--write-kubeconfig-mode=644"]
  variables:
    K3S_TOKEN: "e2e-test-token"
    KUBECONFIG: "/tmp/k3s-kubeconfig.yaml"
    PROVIDER_IMAGE: "${PROVIDER_REGISTRY}/provider-america:${PROVIDER_IMAGE_TAG}"
    MOCK_AMERICA_IMAGE: "${PROVIDER_REGISTRY}/mock-america:${PROVIDER_IMAGE_TAG}"
  before_script:
    # Wait for k3s service to be available and copy kubeconfig
    - |
      echo "Waiting for k3s API server..."
      for i in $(seq 1 60); do
        if wget -qO- --no-check-certificate "https://k3s:6443/readyz" 2>/dev/null; then
          echo "k3s API server is ready"
          break
        fi
        echo "  attempt $i/60..."
        sleep 5
      done
    # Create kubeconfig pointing to the k3s service
    - |
      cat > "${KUBECONFIG}" <<KUBEEOF
      apiVersion: v1
      kind: Config
      clusters:
        - cluster:
            server: https://k3s:6443
            insecure-skip-tls-verify: true
          name: default
      contexts:
        - context:
            cluster: default
            user: default
          name: default
      current-context: default
      users:
        - name: default
          user:
            token: ${K3S_TOKEN}
      KUBEEOF
    - kubectl cluster-info
  script:
    - chmod +x test/e2e/run-e2e.sh
    - ./test/e2e/run-e2e.sh
  timeout: 15m
  needs:
    - build-provider
```

- [ ] **Step 2: Commit**

```bash
git add .gitlab-ci.yml
git commit -m "ci: Add GitLab CI pipeline with build, test, and k3s E2E integration"
```

---

## Task 10: Final verification

- [ ] **Step 1: Run all Go tests**

```bash
go test ./... -v -count=1
```

Expected: all tests pass

- [ ] **Step 2: Build the binary**

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /dev/null ./cmd/provider
```

Expected: clean build

- [ ] **Step 3: Verify CI file syntax**

Validate that `.gitlab-ci.yml` is valid YAML:

```bash
go run golang.org/x/tools/cmd/goyaml@latest < .gitlab-ci.yml || python3 -c "import yaml; yaml.safe_load(open('.gitlab-ci.yml'))"
```

- [ ] **Step 4: Final commit if any cleanup is needed**

```bash
git status
# If clean, no commit needed
```
