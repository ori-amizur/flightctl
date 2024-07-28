package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
		systemdClient: client.NewSystemd(exec),
		exec:          exec,
		log:           log,
	}, nil
}

// Run starts the hook manager and listens for events.
func (m *hookManager) Run(ctx context.Context) {
	m.initialize()
	defer func() {
		if err := m.watcher.Close(); err != nil {
			m.log.Errorf("Error closing watcher: %v", err)
		}
		m.log.Infof("Hook manager stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			m.log.Infof("ctx done")
			return
		case event, ok := <-m.watcher.Events:
			if !ok {
				m.log.Debug("Watcher events channel closed")
				return
			}
			err := m.Handle(ctx, event)
			if err != nil {
				m.log.Errorf("error: %v", err)
			}
		case err, ok := <-m.watcher.Errors:
			if !ok {
				m.log.Debug("Watcher errors channel closed")
				return
			}
			m.log.Errorf("error: %v", err)
		}
	}

}

// Update the manager with the new hook if appropriate.
func (m *hookManager) Update(hook *v1alpha1.DeviceConfigHookSpec) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.handlers == nil {
		return false, ErrHookManagerNotInitialized
	}

	if handler, ok := m.handlers[hook.WatchPath]; !ok || !reflect.DeepEqual(hook, handler.DeviceConfigHookSpec) {
		return true, m.addOrReplaceHookHandler(hook)
	}
	return false, nil
}

func (m *hookManager) Handle(ctx context.Context, event fsnotify.Event) error {
	filePath := event.Name
	handler := m.getHandler(filePath)
	if handler == nil {
		// no handler for this event
		return nil
	}

	actions, ok := handler.opActions[event.Op]
	if !ok {
		// handler does not have any actions for this file operation
		return nil
	}

	// actions
	for i := range actions {
		action := actions[i]
		hookActionType, err := action.Discriminator()
		if err != nil {
			return err
		}

		switch hookActionType {
		case SystemdHookActionType:
			if err := m.handleHookActionSystemd(ctx, &action, filePath); err != nil {
				return err
			}
		case ExecutableHookActionType:
			if err := m.handleHookActionExecutable(ctx, &action, filePath); err != nil {
				return err
			}
		default:
			m.log.Errorf("Unknown hook action type: %s", hookActionType)
			continue
		}
	}

	return nil
}

func (m *hookManager) getHandler(eventName string) *HookHandler {
	// check if the event name is a file or directory
	paths := []string{eventName, filepath.Dir(eventName)}
	for _, watchPath := range paths {
		handler, exists := m.handlers[watchPath]
		if exists {
			return &handler
		}
	}
	return nil
}

func (m *hookManager) handleHookActionSystemd(ctx context.Context, action *v1alpha1.ConfigHookAction, filePath string) error {
	configHook, err := action.AsConfigHookActionSystemdSpec()
	if err != nil {
		return err
	}
	actionTimeout, err := parseTimeout(configHook.Timeout)
	if err != nil {
		return err
	}

	var unitName string
	if configHook.Unit.Name != "" {
		unitName = configHook.Unit.Name
	} else {
		// attempt to extract the systemd unit name from the file path
		unitName, err = getSystemdUnitNameFromFilePath(filePath)
		if err != nil {
			m.log.Errorf("%v: skipping...", err)
			return nil
		}
	}

	for _, op := range configHook.Unit.Operations {
		if err := executeSystemdOperation(ctx, m.systemdClient, op, actionTimeout, unitName); err != nil {
			return err
		}
	}

	return nil
}

func executeSystemdOperation(ctx context.Context, systemdClient *client.Systemd, op v1alpha1.ConfigHookActionSystemdUnitOperations, timeout time.Duration, unitName string) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	switch op {
	case v1alpha1.SystemdStart:
		if err := systemdClient.Start(ctx, unitName); err != nil {
			return err
		}
	case v1alpha1.SystemdStop:
		if err := systemdClient.Stop(ctx, unitName); err != nil {
			return err
		}
	case v1alpha1.SystemdRestart:
		if err := systemdClient.Restart(ctx, unitName); err != nil {
			return err
		}
	case v1alpha1.SystemdReload:
		if err := systemdClient.Reload(ctx, unitName); err != nil {
			return err
		}
	case v1alpha1.SystemdEnable:
		if err := systemdClient.Enable(ctx, unitName); err != nil {
			return err
		}
	case v1alpha1.SystemdDisable:
		if err := systemdClient.Disable(ctx, unitName); err != nil {
			return err
		}
	case v1alpha1.SystemdDaemonReload:
		if err := systemdClient.DaemonReload(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (m *hookManager) handleHookActionExecutable(ctx context.Context, action *v1alpha1.ConfigHookAction, filePath string) error {
	configHook, err := action.AsConfigHookActionExecutableSpec()
	if err != nil {
		return err
	}

	actionTimeout, err := parseTimeout(configHook.Timeout)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	dirExists, err := dirExists(configHook.Executable.WorkDir)
	if err != nil {
		return err
	}

	// we expect the directory to exist should be created by config if its new.
	if !dirExists {
		return os.ErrNotExist
	}

	// replace file token in args if it exists
	tokenMap := newTokenMap(filePath)
	args := replaceTokensInArgs(configHook.Executable.Args, tokenMap)
	_, stderr, exitCode := m.exec.ExecuteWithContextFromDir(ctx, configHook.Executable.WorkDir, configHook.Executable.Path, args...)
	if exitCode != 0 {
		return fmt.Errorf("failed to execute command: %s %d: %s", configHook.Executable.Path, exitCode, stderr)
	}

	return nil
}

func (m *hookManager) addOrReplaceHookHandler(hook *v1alpha1.DeviceConfigHookSpec) error {
	// build lookup for file operations
	opActions := make(map[fsnotify.Op][]v1alpha1.ConfigHookAction)
	for _, action := range hook.Actions {
		hookActionType, err := action.Discriminator()
		if err != nil {
			return err
		}
		switch hookActionType {
		case SystemdHookActionType:
			configHook, err := action.AsConfigHookActionSystemdSpec()
			if err != nil {
				return err
			}
			for _, op := range configHook.TriggerOn {
				opActions[fileOperationToFsnotifyOp(op)] = append(opActions[fileOperationToFsnotifyOp(op)], action)
			}
		case ExecutableHookActionType:
			configHook, err := action.AsConfigHookActionExecutableSpec()
			if err != nil {
				return err
			}
			for _, op := range configHook.TriggerOn {
				opActions[fileOperationToFsnotifyOp(op)] = append(opActions[fileOperationToFsnotifyOp(op)], action)
			}
		default:
			return fmt.Errorf("unknown hook action type: %s", hookActionType)
		}
	}

	// TODO: this is a fair amount of work to do on every update, we should consider optimizing this.
	m.handlers[hook.WatchPath] = HookHandler{
		DeviceConfigHookSpec: hook,
		opActions:            opActions,
	}

	// watcher will error if the path is already being watched
	for _, watchPath := range m.watcher.WatchList() {
		if watchPath == hook.WatchPath {
			return nil
		}
	}

	if err := m.watcher.Add(hook.WatchPath); err != nil {
		return fmt.Errorf("failed adding watch: %w", err)
	}

	return nil
}

func (m *hookManager) initialize() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// initialize the handlers map here for testing observability.
	m.handlers = make(map[string]HookHandler)
}

func (m *hookManager) ResetDefaults() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, watchPath := range m.watcher.WatchList() {
		if err := m.watcher.Remove(watchPath); err != nil {
			return err
		}
	}
	m.handlers = make(map[string]HookHandler)
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

func parseTimeout(timeout *string) (time.Duration, error) {
	if timeout == nil {
		return DefaultHookActionTimeout, nil
	}
	return time.ParseDuration(*timeout)
}

// getSystemdUnitNameFromFilePath attempts to extract the systemd unit name from
// the file path or returns an error if the file does not have a valid systemd
// file suffix.
func getSystemdUnitNameFromFilePath(filePath string) (string, error) {
	unitName := filepath.Base(filePath)

	// list of valid systemd unit file extensions from systemd documentation
	// ref. https://www.freedesktop.org/software/systemd/man/systemd.unit.html
	validExtensions := []string{
		".service",   // Service unit
		".socket",    // Socket unit
		".device",    // Device unit
		".mount",     // Mount unit
		".automount", // Automount unit
		".swap",      // Swap unit
		".target",    // Target unit
		".path",      // Path unit
		".timer",     // Timer unit
		".slice",     // Slice unit
		".scope",     // Scope unit
	}

	// Check if the unit name ends with a valid extension
	for _, ext := range validExtensions {
		if strings.HasSuffix(unitName, ext) {
			return unitName, nil
		}
	}

	return "", fmt.Errorf("invalid systemd unit file: %s", filePath)
}
