package status

import (
	"context"
	"errors"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/internal/agent/device/config"
	"github.com/flightctl/flightctl/internal/util"
	"github.com/flightctl/flightctl/pkg/log"
)

var _ Exporter = (*Hooks)(nil)

// Hooks collects config hook status.
type Hooks struct {
	manager config.HookManager
	log     *log.PrefixLogger
}

func newHooks(log *log.PrefixLogger, manager config.HookManager) *Hooks {
	return &Hooks{
		manager: manager,
		log:     log,
	}
}

// Export returns the status of the config hooks.
func (s *Hooks) Export(ctx context.Context, status *v1alpha1.DeviceStatus) error {
	errs := s.manager.HandleErrors()
	if len(errs) > 0{
		status.Config.PostHooks.Summary.Status = v1alpha1.DeviceConfigPostHooksStatusDegraded
		status.Config.PostHooks.Summary.Info = util.StrToPtr(errors.Join(errs...).Error())
	} else {
		status.Config.PostHooks.Summary.Status = v1alpha1.DeviceConfigPostHooksStatusOnline
		status.Config.PostHooks.Summary.Info = nil
	}
	return nil
}

func (s *Hooks) SetProperties(spec *v1alpha1.RenderedDeviceSpec) {
}
