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
	c.log.Debug("Syncing device configuration")
	defer c.log.Debug("Finished syncing device configuration")

	if desired.Config.PostHooks == nil {
		c.log.Debug("Device resources are nil")
		// Reset all resource alerts to default
		if err := c.hookManager.ResetDefaults(); err != nil {
			return err
		}
	} else {
		// order is important here install new hooks before applying config data
		// so they can be consumed.
		if err := c.ensurePostHooks(desired.Config.PostHooks); err != nil {
			return err
		}
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

func (c *Controller) ensurePostHooks(hooks *[]v1alpha1.DeviceConfigHookSpec) error {
	newWatchPaths := make(map[string]struct{})
	for i := range *hooks {
		hook := (*hooks)[i]
		newWatchPaths[hook.WatchPath] = struct{}{}
		updated, err := c.hookManager.Update(&hook)
		if err != nil {
			return err
		}
		if updated {
			c.log.Infof("Updated hook: %s", hook.Name)
		}
	}

	existingWatchPaths := c.hookManager.WatchList()
	for _, existingWatchPath := range existingWatchPaths {
		if _, ok := newWatchPaths[existingWatchPath]; !ok {
			if err := c.hookManager.WatchRemove(existingWatchPath); err != nil {
				return err
			}
			c.log.Infof("Removed watch: %s", existingWatchPath)
		}
	}

	return nil
}
