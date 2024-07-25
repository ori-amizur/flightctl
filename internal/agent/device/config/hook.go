package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"time"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/internal/agent/client"
	"github.com/flightctl/flightctl/pkg/executer"
	"github.com/flightctl/flightctl/pkg/log"
	"github.com/fsnotify/fsnotify"
)

const (
	DefaultHookActionTimeout = 10 * time.Second
)

var _ HookManager = (*hookManager)(nil)

type hookManager struct {
	mu            sync.Mutex
	watcher       *fsnotify.Watcher
	handlers      map[string]HookHandler
	systemdClient *client.Systemd
	exec          executer.Executer

	log *log.PrefixLogger
}

// NewHookManager creates a new device action manager.
func NewHookManager(log *log.PrefixLogger, exec executer.Executer) (HookManager, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &hookManager{
		watcher:       watcher,
		handlers:      make(map[string]HookHandler),
		systemdClient: client.NewSystemd(exec),
		exec:          exec,
		log:           log,
	}, nil
}

// Run starts the hook manager and listens for events.
func (m *hookManager) Run(ctx context.Context) {
	defer func() {
		if err := m.watcher.Close(); err != nil {
			m.log.Errorf("Error closing watcher: %v", err)
		}
		m.log.Infof("Hook manager stopped")
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

// Update the manager with the new hook if appropriate.
func (m *hookManager) Update(hook *v1alpha1.DeviceConfigHook) (bool, error) {
	handler, ok := m.handlers[hook.FileWatchPath]
	if !ok {
		m.addOrReplaceHookHandler(hook)
		return true, nil
	}

	if reflect.DeepEqual(hook, handler.DeviceConfigHook) {
		return false, nil
	}

	m.addOrReplaceHookHandler(hook)

	return false, nil
}

func (h *hookManager) handleHookActionSystemd(ctx context.Context, configHook *v1alpha1.ConfigHookActionSystemd) error {
	switch *configHook.Action {
	case v1alpha1.ConfigHookActionSystemdStart:
		if err := h.systemdClient.Start(ctx, *configHook.UnitName); err != nil {
			return err
		}
	case v1alpha1.ConfigHookActionSystemdStop:
		if err := h.systemdClient.Stop(ctx, *configHook.UnitName); err != nil {
			return err
		}
	case v1alpha1.ConfigHookActionSystemdRestart:
		if err := h.systemdClient.Restart(ctx, *configHook.UnitName); err != nil {
			return err
		}
	case v1alpha1.ConfigHookActionSystemdReload:
		if err := h.systemdClient.Reload(ctx, *configHook.UnitName); err != nil {
			return err
		}
	case v1alpha1.ConfigHookActionSystemdEnable:
		if err := h.systemdClient.Enable(ctx, *configHook.UnitName); err != nil {
			return err
		}
	case v1alpha1.ConfigHookActionSystemdDisable:
		if err := h.systemdClient.Disable(ctx, *configHook.UnitName); err != nil {
			return err
		}
	}

	return nil
}

func (m *hookManager) handleHookActionExecutable(ctx context.Context, configHook *v1alpha1.ConfigHookActionExecutable) error {
	dirExists, err := dirExists(configHook.WorkingDirectory)
	if err == nil {
		return err
	}

	// we expect the directory to exist should be created by config if its new.
	if !dirExists {
		return os.ErrNotExist
	}

	_, stderr, exitCode := m.exec.ExecuteWithContextFromDir(ctx, configHook.WorkingDirectory, *configHook.ExecutablePath, *configHook.ExecutableArgs...)
	if exitCode != 0 {
		return fmt.Errorf("failed to execute command: %s %d: %s", *configHook.ExecutablePath, exitCode, stderr)
	}

	return nil
}

func (m *hookManager) Handle(event fsnotify.Event) error {
	watchPath := filepath.Dir(event.Name)
	handler, exists := m.handlers[watchPath]
	if !exists {
		return nil
	}

	if _, ok := handler.FileOpLookup[event.Op]; !ok {
		return nil
	}

	// actions
	for _, action := range handler.Actions {
		hookActionType, err := action.Discriminator()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), DefaultHookActionTimeout)
		defer cancel()

		switch hookActionType {
		case SystemdHookActionType:
			configHook, err := action.AsConfigHookActionSystemd()
			if err != nil {
				return err
			}
			if err := m.handleHookActionSystemd(ctx, &configHook); err != nil {
				return err
			}
			m.log.Infof("Added systemd hook for unit: %s", *configHook.UnitName)
		case ExecutableHookActionType:
			configHook, err := action.AsConfigHookActionExecutable()
			if err != nil {
				return err
			}
			if err := m.handleHookActionExecutable(ctx, &configHook); err != nil {
				return err
			}
		default:
			m.log.Errorf("Unknown hook action type: %s", hookActionType)
			continue
		}
	}

	return nil
}

func (m *hookManager) addOrReplaceHookHandler(hook *v1alpha1.DeviceConfigHook) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// build lookup for file operations
	lookup := make(map[fsnotify.Op]struct{})
	for _, op := range hook.FileOperations {
		lookup[fileOperationToFsnotifyOp(op)] = struct{}{}
	}

	m.handlers[hook.FileWatchPath] = HookHandler{
		DeviceConfigHook: hook,
		FileOpLookup:     lookup,
	}

	if err := m.watcher.Add(hook.FileWatchPath); err != nil {
		m.log.Errorf("Error adding watch: %v", err)
	}
}

func (m *hookManager) ResetDefaults() error {
	return nil
}

func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
