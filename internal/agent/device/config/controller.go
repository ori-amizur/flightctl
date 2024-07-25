package config

import (
	"fmt"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/internal/agent/device/fileio"
	"github.com/flightctl/flightctl/pkg/log"
)

// Config controller is responsible for ensuring the device configuration is reconciled
// against the device spec.
type Controller struct {
	hookManager  *HookManager
	deviceWriter *fileio.Writer
	log          *log.PrefixLogger
}

// NewController creates a new config controller.
func NewController(
	hookManager *HookManager,
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
	c.log.Debug("Syncing device configuration")
	defer c.log.Debug("Finished syncing device configuration")

	if desired.Config == nil {
		c.log.Debug("Device config is nil")
		return nil
	}

	if desired.Config.Data != nil {
		desiredConfigRaw := []byte(*desired.Config.Data)
		ignitionConfig, err := ParseAndConvertConfig(desiredConfigRaw)
		if err != nil {
			return fmt.Errorf("parsing and converting config failed: %w", err)
		}

		err = c.deviceWriter.WriteIgnitionFiles(ignitionConfig.Storage.Files...)
		if err != nil {
			return fmt.Errorf("writing ignition files failed: %w", err)
		}
	}

	if desired.Config.Hooks != nil {
		// TODO; implement hooks
		c.log.Warn("Hooks are not implemented")
	}

	return nil
}
