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
