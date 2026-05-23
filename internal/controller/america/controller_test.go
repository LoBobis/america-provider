package america

import (
	"fmt"
	"testing"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
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
			if diff := cmp.Diff(wantErr, err.Error()); diff != "" {
				t.Errorf("CastManaged() error: -want, +got:\n%s", diff)
			}
		})
	}
}

// Compile-time check: all configs produce a common.Deployable.
var _ common.Deployable = TopicConfig().NewObject()
var _ common.Deployable = DataPowerConfig().NewObject()
var _ common.Deployable = ServiceConfig().NewObject()
