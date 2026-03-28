# provider-america

`provider-america` is a minimal [Crossplane](https://crossplane.io/) Provider
that is meant to be used as a america for implementing new Providers. It comes
with the following features that are meant to be refactored:

- A `ProviderConfig` type that only points to a credentials `Secret`.
- A `MyType` resource type that serves as an example managed resource.
- A managed resource controller that reconciles `MyType` objects and simply
  prints their configuration in its `Observe` method.

## Generating a New Resource Kind

Use the built-in code generator to scaffold a new managed resource. It generates
the types file, controller, deepcopy/managed/managedlist methods, and registers
the controller automatically.

```shell
go run cmd/generate/main.go \
  --kind=<KindName> \
  --group=<api-group> \
  --resource-type=<AmericaResourceType> \
  --properties="<Field1:type,Field2:type,...>" \
  --calculation="<Field1:type,Field2:type,...>"
```

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--kind` | Yes | — | PascalCase name of the new kind (e.g. `DataPower`, `Redis`) |
| `--group` | No | `middleware` | API group name (e.g. `middleware`, `storage`, `compute`) |
| `--resource-type` | No | `<Kind>_1_0` | America resource type identifier |
| `--properties` | No | — | Comma-separated `Name:type` pairs for the resource properties |
| `--calculation` | No | — | Comma-separated `Name:type` pairs for computed/read-only fields |

### Examples

Generate a new middleware resource:
```shell
go run cmd/generate/main.go \
  --kind=DataPower \
  --group=middleware \
  --resource-type=DataPowerAdvanced1_0 \
  --properties="Name:string,Port:int,Memory:uint,EnableTLS:bool" \
  --calculation="Status:string"
```

Generate a resource in a new API group (creates the group directory automatically):
```shell
go run cmd/generate/main.go \
  --kind=S3Account \
  --group=storage \
  --resource-type=S3Account_1_0 \
  --properties="BucketName:string,Region:string,SizeGB:int"
```

### What Gets Generated

For an existing group (e.g. `--group=middleware`):
- `apis/middleware/v1alpha1/<kind>_types.go` — CRD types with Deployable interface
- `internal/controller/<kind>/<kind>.go` — Controller with webhook support
- Appends to `apis/middleware/v1alpha1/zz_generated.deepcopy.go`
- Appends to `apis/middleware/v1alpha1/zz_generated.managed.go`
- Appends to `apis/middleware/v1alpha1/zz_generated.managedlist.go`
- Patches `internal/controller/america.go` to register the new controller

For a **new** group (e.g. `--group=storage`):
- All of the above, plus:
- `apis/storage/v1alpha1/groupversion_info.go` — Group/version registration
- `apis/storage/v1alpha1/common_status.go` — Shared AtProviderStatus types
- `apis/storage/v1alpha1/zz_generated.*.go` — Base generated files
- Patches `apis/america.go` to register the new group's scheme

After generating, verify with:
```shell
go build ./...
```

## Developing

1. Use this repository as a template to create a new one.
1. Run `make submodules` to initialize the "build" Make submodule we use for CI/CD.
1. Generate new resource kinds using the generator (see above).
1. Run `make reviewable` to run code generation, linters, and tests.
1. Run `make build` to build the provider.

Refer to Crossplane's [CONTRIBUTING.md] file for more information on how the
Crossplane community prefers to work. The [Provider Development][provider-dev]
guide may also be of use.

[CONTRIBUTING.md]: https://github.com/crossplane/crossplane/blob/master/CONTRIBUTING.md
[provider-dev]: https://github.com/crossplane/crossplane/blob/master/contributing/guide-provider-development.md
