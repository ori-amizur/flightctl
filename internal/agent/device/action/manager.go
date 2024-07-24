package action

import (
	"context"
	"path/filepath"

	"github.com/flightctl/flightctl/pkg/log"
	"github.com/fsnotify/fsnotify"
)

type Manager struct {
	watcher  *fsnotify.Watcher
	handlers map[string][]Handler

	log *log.PrefixLogger
}

// NewManager creates a new device action manager.
func NewManager(log *log.PrefixLogger) (*Manager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Manager{
		watcher:  watcher,
		handlers: make(map[string][]Handler),
		log:      log,
	}, nil
}

// Run starts the manager and listens for events.
func (a *Manager) Run(ctx context.Context) {
	defer func() {
		if err := a.watcher.Close(); err != nil {
			a.log.Errorf("Error closing watcher: %v", err)
		}
		a.log.Infof("Action manager stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-a.watcher.Events:
			if !ok {
				return
			}
			err := a.Handle(event)
			if err != nil {
				a.log.Errorf("error: %v", err)
			}
		case err, ok := <-a.watcher.Errors:
			if !ok {
				return
			}
			a.log.Errorf("error: %v", err)
		}
	}
}

func (a *Manager) Handle(event fsnotify.Event) error {
	watchPath := filepath.Dir(event.Name)
	handlers, exists := a.handlers[watchPath]
	if !exists {
		return nil
	}

	for _, handle := range handlers {
		if err := handle.EventCallbackFn(event); err != nil {
			return err
		}
	}

	return nil
}
