// cmd/generate/main.go
//
// Usage:
//
//	go run cmd/generate/main.go \
//	  --kind=DataPower \
//	  --group=middleware \
//	  --resource-type=DataPowerAdvanced1_0 \
//	  --properties="Name:string,Port:int,Memory:uint,EnableTLS:bool" \
//	  --calculation="Status:string,LastSyncTime:string"
//
// This generates:
//   - apis/{group}/v1alpha1/{kind}_types.go
//   - internal/controller/{kind}/{kind}.go
//
// And appends to:
//   - apis/{group}/v1alpha1/zz_generated.deepcopy.go
//   - apis/{group}/v1alpha1/zz_generated.managed.go
//   - apis/{group}/v1alpha1/zz_generated.managedlist.go
//   - internal/controller/register.go
//
// If the group directory doesn't exist, it also creates:
//   - apis/{group}/v1alpha1/groupversion_info.go
//   - apis/{group}/v1alpha1/common_status.go (copies from middleware)
// And registers the new group's scheme in apis/template.go

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"
)

type Field struct {
	Name    string
	Type    string
	JSONTag string
}

type KindConfig struct {
	Kind            string // e.g. "DataPower"
	KindLower       string // e.g. "datapower"
	KindReceiver    string // e.g. "d" (first letter lowercase)
	ResourceType    string // e.g. "DataPowerAdvanced1_0"
	Group           string // e.g. "middleware", "storage", "compute"
	GroupDomain     string // e.g. "middleware.provideramerica.crossplane.io"
	Properties      []Field
	Calculations    []Field
	HasCalculations bool
}

func main() {
	cfg := parseArgs()

	root := findProjectRoot()

	ensureGroupDir(root, cfg)
	generateTypes(root, cfg)
	generateController(root, cfg)
	appendDeepCopy(root, cfg)
	appendManaged(root, cfg)
	appendManagedList(root, cfg)
	patchRegister(root, cfg)

	fmt.Printf("\nGenerated kind %q in group %q successfully!\n", cfg.Kind, cfg.Group)
	fmt.Println("\nFiles created:")
	fmt.Printf("  apis/%s/v1alpha1/%s_types.go\n", cfg.Group, cfg.KindLower)
	fmt.Printf("  internal/controller/%s/%s.go\n", cfg.KindLower, cfg.KindLower)
	fmt.Println("\nFiles updated:")
	fmt.Printf("  apis/%s/v1alpha1/zz_generated.deepcopy.go\n", cfg.Group)
	fmt.Printf("  apis/%s/v1alpha1/zz_generated.managed.go\n", cfg.Group)
	fmt.Printf("  apis/%s/v1alpha1/zz_generated.managedlist.go\n", cfg.Group)
	fmt.Println("  internal/controller/register.go")
	fmt.Println("\nDon't forget to run: go build ./...")
}

func parseArgs() KindConfig {
	var kind, group, resourceType, propsStr, calcStr string

	for _, arg := range os.Args[1:] {
		switch {
		case strings.HasPrefix(arg, "--kind="):
			kind = strings.TrimPrefix(arg, "--kind=")
		case strings.HasPrefix(arg, "--group="):
			group = strings.TrimPrefix(arg, "--group=")
		case strings.HasPrefix(arg, "--resource-type="):
			resourceType = strings.TrimPrefix(arg, "--resource-type=")
		case strings.HasPrefix(arg, "--properties="):
			propsStr = strings.TrimPrefix(arg, "--properties=")
		case strings.HasPrefix(arg, "--calculation="):
			calcStr = strings.TrimPrefix(arg, "--calculation=")
		}
	}

	if kind == "" {
		fmt.Println("Usage: go run cmd/generate/main.go --kind=MyKind [--group=middleware] --resource-type=MyType1_0 --properties=\"Name:string,Port:int\" [--calculation=\"Field:type\"]")
		os.Exit(1)
	}

	if group == "" {
		group = "middleware"
	}

	if resourceType == "" {
		resourceType = kind + "_1_0"
	}

	cfg := KindConfig{
		Kind:         kind,
		KindLower:    strings.ToLower(kind),
		KindReceiver: strings.ToLower(kind[:1]),
		ResourceType: resourceType,
		Group:        group,
		GroupDomain:  group + ".provideramerica.crossplane.io",
		Properties:   parseFields(propsStr),
		Calculations: parseFields(calcStr),
	}
	cfg.HasCalculations = len(cfg.Calculations) > 0

	return cfg
}

func parseFields(s string) []Field {
	if s == "" {
		return nil
	}
	var fields []Field
	for _, pair := range strings.Split(s, ",") {
		parts := strings.SplitN(pair, ":", 2)
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid field format: %q (expected Name:type)\n", pair)
			os.Exit(1)
		}
		fields = append(fields, Field{
			Name:    parts[0],
			Type:    parts[1],
			JSONTag: camelToSnake(parts[0]),
		})
	}
	return fields
}

func camelToSnake(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, unicode.ToLower(r))
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}

func findProjectRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			fmt.Fprintln(os.Stderr, "Cannot find go.mod in parent directories")
			os.Exit(1)
		}
		dir = parent
	}
}

func ensureGroupDir(root string, cfg KindConfig) {
	groupDir := filepath.Join(root, "apis", cfg.Group, "v1alpha1")

	// Check if the group directory already exists
	if _, err := os.Stat(groupDir); err == nil {
		return // already exists
	}

	fmt.Printf("  Creating new API group: %s\n", cfg.Group)
	os.MkdirAll(groupDir, 0755)

	// Create groupversion_info.go
	gvPath := filepath.Join(groupDir, "groupversion_info.go")
	writeTemplate(gvPath, groupVersionTemplate, cfg)

	// Create common_status.go (same shared status types)
	csPath := filepath.Join(groupDir, "common_status.go")
	writeTemplate(csPath, commonStatusTemplate, cfg)

	// Create empty zz_generated files with correct package header
	deepCopyPath := filepath.Join(groupDir, "zz_generated.deepcopy.go")
	writeTemplate(deepCopyPath, emptyDeepCopyTemplate, cfg)

	managedPath := filepath.Join(groupDir, "zz_generated.managed.go")
	writeTemplate(managedPath, emptyManagedTemplate, cfg)

	managedListPath := filepath.Join(groupDir, "zz_generated.managedlist.go")
	writeTemplate(managedListPath, emptyManagedListTemplate, cfg)

	// Patch apis/template.go to register the new group scheme
	patchTemplateScheme(root, cfg)
}

func patchTemplateScheme(root string, cfg KindConfig) {
	path := filepath.Join(root, "apis", "template.go")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read %s: %v\n", path, err)
		os.Exit(1)
	}

	content := string(data)
	alias := cfg.Group + "v1alpha1"
	importLine := fmt.Sprintf("\t%s \"github.com/crossplane/provider-america/apis/%s/v1alpha1\"", alias, cfg.Group)
	schemeLine := fmt.Sprintf("\t\t%s.SchemeBuilder.AddToScheme,", alias)

	// Check if already registered
	if strings.Contains(content, importLine) {
		return
	}

	// Add import — insert before the closing paren of the import block
	firstImportClose := strings.Index(content, ")")
	if firstImportClose == -1 {
		fmt.Fprintln(os.Stderr, "Cannot find import closing paren in template.go")
		os.Exit(1)
	}
	content = content[:firstImportClose] + importLine + "\n" + content[firstImportClose:]

	// Add scheme registration — insert before the closing paren of AddToSchemes append
	// Find the last occurrence of ".AddToScheme," and add after it
	lastScheme := strings.LastIndex(content, ".SchemeBuilder.AddToScheme,")
	if lastScheme == -1 {
		fmt.Fprintln(os.Stderr, "Cannot find AddToScheme in template.go")
		os.Exit(1)
	}
	// Find end of that line
	eol := strings.Index(content[lastScheme:], "\n")
	insertPos := lastScheme + eol
	content = content[:insertPos] + "\n" + schemeLine + content[insertPos:]

	os.WriteFile(path, []byte(content), 0644)
	fmt.Printf("  template.go: registered %s group scheme\n", cfg.Group)
}

func generateTypes(root string, cfg KindConfig) {
	path := filepath.Join(root, "apis", cfg.Group, "v1alpha1", cfg.KindLower+"_types.go")
	writeTemplate(path, typesTemplate, cfg)
}

func generateController(root string, cfg KindConfig) {
	dir := filepath.Join(root, "internal", "controller", cfg.KindLower)
	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, cfg.KindLower+".go")
	writeTemplate(path, controllerTemplate, cfg)
}

func appendDeepCopy(root string, cfg KindConfig) {
	path := filepath.Join(root, "apis", cfg.Group, "v1alpha1", "zz_generated.deepcopy.go")
	appendTemplate(path, deepCopyTemplate, cfg)
}

func appendManaged(root string, cfg KindConfig) {
	path := filepath.Join(root, "apis", cfg.Group, "v1alpha1", "zz_generated.managed.go")
	appendTemplate(path, managedTemplate, cfg)
}

func appendManagedList(root string, cfg KindConfig) {
	path := filepath.Join(root, "apis", cfg.Group, "v1alpha1", "zz_generated.managedlist.go")
	appendTemplate(path, managedListTemplate, cfg)
}

func patchRegister(root string, cfg KindConfig) {
	path := filepath.Join(root, "internal", "controller", "register.go")
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read %s: %v\n", path, err)
		os.Exit(1)
	}

	content := string(data)
	importLine := fmt.Sprintf("\t\"github.com/crossplane/provider-america/internal/controller/%s\"", cfg.KindLower)
	setupLine := fmt.Sprintf("\t\t%s.SetupGated,", cfg.KindLower)

	// Check if already registered
	if strings.Contains(content, importLine) {
		fmt.Printf("  register.go: %s already registered, skipping\n", cfg.Kind)
		return
	}

	// Add import — find the last controller import and insert after it
	lastImport := strings.LastIndex(content, "\"github.com/crossplane/provider-america/internal/controller/")
	if lastImport == -1 {
		fmt.Fprintln(os.Stderr, "Cannot find controller imports in register.go")
		os.Exit(1)
	}
	eolImport := strings.Index(content[lastImport:], "\n")
	insertImportPos := lastImport + eolImport
	content = content[:insertImportPos] + "\n" + importLine + content[insertImportPos:]

	// Add to America controllers setup list — find the last .SetupGated, line and insert after it
	lastSetup := strings.LastIndex(content, ".SetupGated,")
	if lastSetup == -1 {
		fmt.Fprintln(os.Stderr, "Cannot find setup list in register.go")
		os.Exit(1)
	}
	eolSetup := strings.Index(content[lastSetup:], "\n")
	insertSetupPos := lastSetup + eolSetup
	content = content[:insertSetupPos] + "\n" + setupLine + content[insertSetupPos:]

	os.WriteFile(path, []byte(content), 0644)
	fmt.Printf("  register.go: added %s controller\n", cfg.Kind)
}

func writeTemplate(path string, tmplStr string, cfg KindConfig) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot create %s: %v\n", path, err)
		os.Exit(1)
	}
	defer f.Close()

	tmpl := template.Must(template.New("").Parse(tmplStr))
	if err := tmpl.Execute(f, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Template error for %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("  Created: %s\n", path)
}

func appendTemplate(path string, tmplStr string, cfg KindConfig) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot open %s: %v\n", path, err)
		os.Exit(1)
	}
	defer f.Close()

	tmpl := template.Must(template.New("").Parse(tmplStr))
	if err := tmpl.Execute(f, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Template error for %s: %v\n", path, err)
		os.Exit(1)
	}
	fmt.Printf("  Appended to: %s\n", path)
}

// ─── Templates ───────────────────────────────────────────────────────────────

var typesTemplate = `package v1alpha1

import (
	"reflect"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	xpv2 "github.com/crossplane/crossplane-runtime/v2/apis/common/v2"
)

type {{.Kind}}Properties struct {
{{- range .Properties}}
	{{.Name}} {{.Type}} ` + "`" + `json:"{{.JSONTag}}"` + "`" + `
{{- end}}
}

type {{.Kind}}Calculation struct {
{{- range .Calculations}}
	{{.Name}} {{.Type}} ` + "`" + `json:"{{.JSONTag}}"` + "`" + `
{{- end}}
}

// {{.Kind}}Parameters are the configurable fields of a {{.Kind}}.
type {{.Kind}}Parameters struct {
	// +kubebuilder:default="{{.ResourceType}}"
	ResourceType            string ` + "`" + `json:"resource_type"` + "`" + `
	{{.Kind}}Properties     ` + "`" + `json:"properties"` + "`" + `
	*{{.Kind}}Calculation   ` + "`" + `json:"resource_info"` + "`" + `
}

// A {{.Kind}}Spec defines the desired state of a {{.Kind}}.
type {{.Kind}}Spec struct {
	xpv2.ManagedResourceSpec ` + "`" + `json:",inline"` + "`" + `
	Region                   string              ` + "`" + `json:"region"` + "`" + `
	Environment              string              ` + "`" + `json:"environment"` + "`" + `
	BundleID                 string              ` + "`" + `json:"bundle_id"` + "`" + `
	ForProvider              {{.Kind}}Parameters  ` + "`" + `json:"forProvider"` + "`" + `
}

// +kubebuilder:object:root=true

// A {{.Kind}} is an America managed resource.
// +kubebuilder:printcolumn:name="READY",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="SYNCED",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="EXTERNAL-NAME",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="AGE",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,categories={crossplane,managed,america}
type {{.Kind}} struct {
	metav1.TypeMeta   ` + "`" + `json:",inline"` + "`" + `
	metav1.ObjectMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `

	Spec   {{.Kind}}Spec  ` + "`" + `json:"spec"` + "`" + `
	Status ResourceStatus ` + "`" + `json:"status,omitempty"` + "`" + `
}

func ({{.KindReceiver}} *{{.Kind}}) GetName() string             { return {{.KindReceiver}}.Name }
func ({{.KindReceiver}} *{{.Kind}}) GetBundleID() string         { return {{.KindReceiver}}.Spec.BundleID }
func ({{.KindReceiver}} *{{.Kind}}) GetResourceType() string     { return {{.KindReceiver}}.Spec.ForProvider.ResourceType }
func ({{.KindReceiver}} *{{.Kind}}) GetEnvironment() string      { return {{.KindReceiver}}.Spec.Environment }
func ({{.KindReceiver}} *{{.Kind}}) GetRegion() string           { return {{.KindReceiver}}.Spec.Region }
func ({{.KindReceiver}} *{{.Kind}}) GetForProvider() interface{} { return &{{.KindReceiver}}.Spec.ForProvider }
func ({{.KindReceiver}} *{{.Kind}}) GetDeploymentID() string     { return {{.KindReceiver}}.Status.AtProvider.DeploymentID }

func ({{.KindReceiver}} *{{.Kind}}) SetDeploymentID(id string) { {{.KindReceiver}}.Status.AtProvider.DeploymentID = id }
func ({{.KindReceiver}} *{{.Kind}}) SetStatus(status string)   { {{.KindReceiver}}.Status.AtProvider.Status = status }
func ({{.KindReceiver}} *{{.Kind}}) SetAdditionalInfo(raw *runtime.RawExtension) {
	{{.KindReceiver}}.Status.AtProvider.AdditionalInfo = raw
}
func ({{.KindReceiver}} *{{.Kind}}) SetCondition(condition xpv1.Condition) { {{.KindReceiver}}.Status.SetConditions(condition) }

// +kubebuilder:object:root=true

// {{.Kind}}List contains a list of {{.Kind}}
type {{.Kind}}List struct {
	metav1.TypeMeta ` + "`" + `json:",inline"` + "`" + `
	metav1.ListMeta ` + "`" + `json:"metadata,omitempty"` + "`" + `
	Items           []{{.Kind}} ` + "`" + `json:"items"` + "`" + `
}

// {{.Kind}} type metadata.
var (
	{{.Kind}}Kind             = reflect.TypeOf({{.Kind}}{}).Name()
	{{.Kind}}GroupKind        = schema.GroupKind{Group: Group, Kind: {{.Kind}}Kind}.String()
	{{.Kind}}KindAPIVersion   = {{.Kind}}Kind + "." + SchemeGroupVersion.String()
	{{.Kind}}GroupVersionKind = SchemeGroupVersion.WithKind({{.Kind}}Kind)
)

func init() {
	SchemeBuilder.Register(&{{.Kind}}{}, &{{.Kind}}List{})
}
`

var controllerTemplate = `package {{.KindLower}}

import (
	"context"

	"github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	xpevent "github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/crossplane/crossplane-runtime/v2/pkg/statemetrics"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	v1alpha1 "github.com/crossplane/provider-america/apis/{{.Group}}/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-america/apis/v1alpha1"
	americaClient "github.com/crossplane/provider-america/internal/clients/america"
	"github.com/crossplane/provider-america/internal/controller/common"
)

const (
	errNot{{.Kind}} = "managed resource is not a {{.Kind}} custom resource"
)

// SetupGated adds a controller that reconciles {{.Kind}} managed resources with safe-start support.
func SetupGated(mgr ctrl.Manager, o controller.Options, webhookEventCh chan event.GenericEvent) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o, webhookEventCh); err != nil {
			panic(errors.Wrap(err, "cannot setup {{.Kind}} controller"))
		}
	}, v1alpha1.{{.Kind}}GroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles {{.Kind}} managed resources.
func Setup(mgr ctrl.Manager, o controller.Options, webhookEventCh chan event.GenericEvent) error {
	name := managed.ControllerName(v1alpha1.{{.Kind}}GroupKind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			logger:             o.Logger,
			kube:               mgr.GetClient(),
			usage:              resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newAmericaClientFn: americaClient.NewClient}),
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
			mgr.GetClient(), o.Logger, o.MetricOptions.MRStateMetrics, &v1alpha1.{{.Kind}}List{}, o.MetricOptions.PollStateMetricInterval,
		)
		if err := mgr.Add(stateMetricsRecorder); err != nil {
			return errors.Wrap(err, "cannot register MR state metrics recorder for kind v1alpha1.{{.Kind}}List")
		}
	}

	r := managed.NewReconciler(mgr, resource.ManagedKind(v1alpha1.{{.Kind}}GroupVersionKind), opts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.{{.Kind}}{}).
		WatchesRawSource(source.Channel(webhookEventCh, &handler.EnqueueRequestForObject{})).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	logger             logging.Logger
	kube               client.Client
	usage              *resource.ProviderConfigUsageTracker
	newAmericaClientFn func(log logging.Logger, creds string) (americaClient.Client, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.{{.Kind}})
	if !ok {
		return nil, errors.New(errNot{{.Kind}})
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
	}, nil
}

type external struct {
	logger         logging.Logger
	americaConfig  apisv1alpha1.AmericaConfig
	kube           client.Client
	america_client americaClient.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.{{.Kind}})
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNot{{.Kind}})
	}

	return common.ObserveExternal(ctx, c.america_client, c.americaConfig, c.kube, cr, errNot{{.Kind}})
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.{{.Kind}})
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNot{{.Kind}})
	}
	return common.CreateExternal(ctx, c.america_client, c.americaConfig, c.kube, cr)
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.{{.Kind}})
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNot{{.Kind}})
	}

	return common.UpdateExternal(ctx, c.america_client, c.americaConfig, c.kube, cr)
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.{{.Kind}})
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNot{{.Kind}})
	}

	return common.DeleteExternal(c.america_client, c.americaConfig, cr)
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
`

var deepCopyTemplate = `
// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *{{.Kind}}) DeepCopyInto(out *{{.Kind}}) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

func (in *{{.Kind}}) DeepCopy() *{{.Kind}} {
	if in == nil {
		return nil
	}
	out := new({{.Kind}})
	in.DeepCopyInto(out)
	return out
}

func (in *{{.Kind}}) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *{{.Kind}}List) DeepCopyInto(out *{{.Kind}}List) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]{{.Kind}}, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

func (in *{{.Kind}}List) DeepCopy() *{{.Kind}}List {
	if in == nil {
		return nil
	}
	out := new({{.Kind}}List)
	in.DeepCopyInto(out)
	return out
}

func (in *{{.Kind}}List) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *{{.Kind}}Properties) DeepCopyInto(out *{{.Kind}}Properties) {
	*out = *in
}

func (in *{{.Kind}}Properties) DeepCopy() *{{.Kind}}Properties {
	if in == nil {
		return nil
	}
	out := new({{.Kind}}Properties)
	in.DeepCopyInto(out)
	return out
}

func (in *{{.Kind}}Calculation) DeepCopyInto(out *{{.Kind}}Calculation) {
	*out = *in
}

func (in *{{.Kind}}Calculation) DeepCopy() *{{.Kind}}Calculation {
	if in == nil {
		return nil
	}
	out := new({{.Kind}}Calculation)
	in.DeepCopyInto(out)
	return out
}

func (in *{{.Kind}}Parameters) DeepCopyInto(out *{{.Kind}}Parameters) {
	*out = *in
	out.{{.Kind}}Properties = in.{{.Kind}}Properties
	if in.{{.Kind}}Calculation != nil {
		in, out := &in.{{.Kind}}Calculation, &out.{{.Kind}}Calculation
		*out = new({{.Kind}}Calculation)
		**out = **in
	}
}

func (in *{{.Kind}}Parameters) DeepCopy() *{{.Kind}}Parameters {
	if in == nil {
		return nil
	}
	out := new({{.Kind}}Parameters)
	in.DeepCopyInto(out)
	return out
}

func (in *{{.Kind}}Spec) DeepCopyInto(out *{{.Kind}}Spec) {
	*out = *in
	in.ManagedResourceSpec.DeepCopyInto(&out.ManagedResourceSpec)
	in.ForProvider.DeepCopyInto(&out.ForProvider)
}

func (in *{{.Kind}}Spec) DeepCopy() *{{.Kind}}Spec {
	if in == nil {
		return nil
	}
	out := new({{.Kind}}Spec)
	in.DeepCopyInto(out)
	return out
}
`

var managedTemplate = `
// GetCondition of this {{.Kind}}.
func (mg *{{.Kind}}) GetCondition(ct xpv1.ConditionType) xpv1.Condition {
	return mg.Status.GetCondition(ct)
}

// GetManagementPolicies of this {{.Kind}}.
func (mg *{{.Kind}}) GetManagementPolicies() xpv1.ManagementPolicies {
	return mg.Spec.ManagementPolicies
}

// GetProviderConfigReference of this {{.Kind}}.
func (mg *{{.Kind}}) GetProviderConfigReference() *xpv1.ProviderConfigReference {
	return mg.Spec.ProviderConfigReference
}

// GetWriteConnectionSecretToReference of this {{.Kind}}.
func (mg *{{.Kind}}) GetWriteConnectionSecretToReference() *xpv1.LocalSecretReference {
	return mg.Spec.WriteConnectionSecretToReference
}

// SetConditions of this {{.Kind}}.
func (mg *{{.Kind}}) SetConditions(c ...xpv1.Condition) {
	mg.Status.SetConditions(c...)
}

// SetManagementPolicies of this {{.Kind}}.
func (mg *{{.Kind}}) SetManagementPolicies(r xpv1.ManagementPolicies) {
	mg.Spec.ManagementPolicies = r
}

// SetProviderConfigReference of this {{.Kind}}.
func (mg *{{.Kind}}) SetProviderConfigReference(r *xpv1.ProviderConfigReference) {
	mg.Spec.ProviderConfigReference = r
}

// SetWriteConnectionSecretToReference of this {{.Kind}}.
func (mg *{{.Kind}}) SetWriteConnectionSecretToReference(r *xpv1.LocalSecretReference) {
	mg.Spec.WriteConnectionSecretToReference = r
}
`

var managedListTemplate = `
// GetItems of this {{.Kind}}List.
func (l *{{.Kind}}List) GetItems() []resource.Managed {
	items := make([]resource.Managed, len(l.Items))
	for i := range l.Items {
		items[i] = &l.Items[i]
	}
	return items
}
`

// ─── Group scaffolding templates ─────────────────────────────────────────────

var groupVersionTemplate = `package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Package type metadata.
const (
	Group   = "{{.GroupDomain}}"
	Version = "v1alpha1"
)

var (
	// SchemeGroupVersion is group version used to register these objects
	SchemeGroupVersion = schema.GroupVersion{Group: Group, Version: Version}

	// SchemeBuilder is used to add go types to the GroupVersionKind scheme
	SchemeBuilder = &scheme.Builder{GroupVersion: SchemeGroupVersion}
)
`

var commonStatusTemplate = `package v1alpha1

import (
	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AtProviderStatus represents the observed state from the America API.
type AtProviderStatus struct {
	DeploymentID   string                ` + "`" + `json:"deploymentID,omitempty"` + "`" + `
	Status         string                ` + "`" + `json:"status,omitempty"` + "`" + `
	AdditionalInfo *runtime.RawExtension ` + "`" + `json:"additionalInfo,omitempty"` + "`" + `
}

// ResourceStatus is the shared status for all America managed resources.
type ResourceStatus struct {
	xpv1.ResourceStatus ` + "`" + `json:",inline"` + "`" + `
	AtProvider          AtProviderStatus ` + "`" + `json:"atProvider,omitempty"` + "`" + `
}
`

var emptyDeepCopyTemplate = `//go:build !ignore_autogenerated

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AtProviderStatus) DeepCopyInto(out *AtProviderStatus) {
	*out = *in
	if in.AdditionalInfo != nil {
		in, out := &in.AdditionalInfo, &out.AdditionalInfo
		*out = new(runtime.RawExtension)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AtProviderStatus.
func (in *AtProviderStatus) DeepCopy() *AtProviderStatus {
	if in == nil {
		return nil
	}
	out := new(AtProviderStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourceStatus) DeepCopyInto(out *ResourceStatus) {
	*out = *in
	in.ResourceStatus.DeepCopyInto(&out.ResourceStatus)
	in.AtProvider.DeepCopyInto(&out.AtProvider)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceStatus.
func (in *ResourceStatus) DeepCopy() *ResourceStatus {
	if in == nil {
		return nil
	}
	out := new(ResourceStatus)
	in.DeepCopyInto(out)
	return out
}
`

var emptyManagedTemplate = `// Code generated by angryjet. DO NOT EDIT.

package v1alpha1

import xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"

// Ensure xpv1 import is used.
var _ = xpv1.ConditionType("")
`

var emptyManagedListTemplate = `// Code generated by angryjet. DO NOT EDIT.

package v1alpha1

import resource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"

// Ensure resource import is used.
var _ resource.Managed = nil
`
