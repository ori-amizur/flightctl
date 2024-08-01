package config

import (
	"fmt"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/samber/lo"
)

// Config controller is responsible for ensuring the device configuration is reconciled
// against the device spec.
type Controller struct {
	hookManager  HookManager
	deviceWriter *fileio.Writer
	log          *log.PrefixLogger
}

// NewController creates a new config controller.
func NewController(
	hookManager HookManager,
	deviceWriter *fileio.Writer,
	log *log.PrefixLogger,
) *Controller {
	return &Controller{
		hookManager:  hookManager,
		deviceWriter: deviceWriter,
		log:          log,
	}
}

func (c *Controller) Sync(desired *v1alpha1.RenderedDeviceSpec) error {
	if err := c.hookManager.EnsurePostHooks(lo.FromPtr(desired.Config.PostHooks)); err != nil {
		return err
	}

	if desired.Config.Data != nil {
		data := *desired.Config.Data
		return c.ensureConfigData(data)
	}

	return nil
}

func (c *Controller) ensureConfigData(data string) error {
	desiredConfigRaw := []byte(data)
	ignitionConfig, err := ParseAndConvertConfig(desiredConfigRaw)
	if err != nil {
		return fmt.Errorf("parsing and converting config failed: %w", err)
	}

	err = c.deviceWriter.WriteIgnitionFiles(ignitionConfig.Storage.Files...)
	if err != nil {
		return fmt.Errorf("writing ignition files failed: %w", err)
	}
	return nil
}
