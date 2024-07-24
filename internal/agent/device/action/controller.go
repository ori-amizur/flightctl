package action

import (
	"context"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/pkg/log"
)

type Controller struct {
	manager *Manager
	log     *log.PrefixLogger
}

// NewController creates a new device action controller.
func NewController(manager *Manager, log *log.PrefixLogger) *Controller {
	return &Controller{
		manager: manager,
		log:     log,
	}
}

func (c *Controller) Sync(ctx context.Context, desired *v1alpha1.RenderedDeviceSpec) error {
	c.log.Debug("Syncing device action controller")
	defer c.log.Debug("Finished syncing device action controller")

	// TODO: need apis :)
	return c.ensureActions()
}

func (c *Controller) ensureActions() error {
	return nil
}
