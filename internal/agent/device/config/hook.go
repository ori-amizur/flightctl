package config

import (
	"context"
	"path/filepath"

	"github.com/flightctl/flightctl/pkg/log"
	"github.com/fsnotify/fsnotify"
)

type HookManager struct {
	watcher  *fsnotify.Watcher
	handlers map[string][]EventHandler

	log *log.PrefixLogger
}

// NewHookManager creates a new device action manager.
func NewHookManager(log *log.PrefixLogger) (*HookManager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &HookManager{
		watcher:  watcher,
		handlers: make(map[string][]EventHandler),
		log:      log,
	}, nil
}

// Run starts the hook manager and listens for events.
func (m *HookManager) Run(ctx context.Context) {
	defer func() {
		if err := m.watcher.Close(); err != nil {
			m.log.Errorf("Error closing watcher: %v", err)
		}
		m.log.Infof("Action manager stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-m.watcher.Events:
			if !ok {
				return
			}
			err := m.Handle(event)
			if err != nil {
				m.log.Errorf("error: %v", err)
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				return
			}
			m.log.Errorf("error: %v", err)
		}
	}
}

func (m *HookManager) Handle(event fsnotify.Event) error {
	watchPath := filepath.Dir(event.Name)
	handlers, exists := m.handlers[watchPath]
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
