package america

import (
	"fmt"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	v1alpha1 "github.com/crossplane/provider-america/apis/middleware/v1alpha1"
	"github.com/crossplane/provider-america/internal/controller/common"
)

func TopicConfig() ResourceConfig {
	return ResourceConfig{
		Kind:             v1alpha1.TopicKind,
		GroupKind:        v1alpha1.TopicGroupKind,
		GroupVersionKind: v1alpha1.TopicGroupVersionKind,
		NewObject:        func() common.Deployable { return &v1alpha1.Topic{} },
		NewObjectList:    func() resource.ManagedList { return &v1alpha1.TopicList{} },
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
		NewObjectList:    func() resource.ManagedList { return &v1alpha1.DataPowerList{} },
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
		NewObjectList:    func() resource.ManagedList { return &v1alpha1.ServiceList{} },
		CastManaged: func(mg resource.Managed) (common.Deployable, error) {
			cr, ok := mg.(*v1alpha1.Service)
			if !ok {
				return nil, fmt.Errorf("managed resource is not a %s custom resource", v1alpha1.ServiceKind)
			}
			return cr, nil
		},
	}
}
