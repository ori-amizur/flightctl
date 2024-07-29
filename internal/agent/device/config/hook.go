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
	watcher       Watcher
	handlers      map[string]HookHandler
	systemdClient *client.Systemd
	exec          executer.Executer

	log *log.PrefixLogger
}

// NewHookManager creates a new device action manager.
func NewHookManager(log *log.PrefixLogger, exec executer.Executer) (HookManager, error) {
	watcher, err := NewInotifyWatcher()
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
		case event, ok := <-m.watcher.Events():
			if !ok {
				m.log.Debug("Watcher events channel closed")
				return
			}
			err := m.Handle(ctx, event)
			if err != nil {
				m.log.Errorf("error: %v", err)
			}
		case err, ok := <-m.watcher.Errors():
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
		return true, addOrReplaceHookHandler(m.watcher, hook, m.handlers)
	}
	return false, nil
}

func (m *hookManager) WatchList() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.watcher.List()
}

func (m *hookManager) WatchRemove(watchPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.handlers == nil {
		return ErrHookManagerNotInitialized
	}

	if _, ok := m.handlers[watchPath]; ok {
		if err := m.watcher.Remove(watchPath); err != nil {
			return fmt.Errorf("failed removing watch: %w", err)
		}
		delete(m.handlers, watchPath)
	}
	return nil
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
			if err := handleHookActionSystemd(ctx, m.log, m.systemdClient, &action, filePath); err != nil {
				return err
			}
		case ExecutableHookActionType:
			action, err := action.AsConfigHookActionExecutableSpec()
			if err != nil {
				return err
			}

			if err := handleHookActionExecutable(ctx, m.exec, &action, filePath); err != nil {
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

func handleHookActionSystemd(ctx context.Context, log *log.PrefixLogger, systemdClient *client.Systemd, action *v1alpha1.ConfigHookAction, filePath string) error {
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
			log.Errorf("%v: skipping...", err)
			return nil
		}
	}

	for _, op := range configHook.Unit.Operations {
		if err := executeSystemdOperation(ctx, systemdClient, op, actionTimeout, unitName); err != nil {
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

func handleHookActionExecutable(ctx context.Context, exec executer.Executer, action *v1alpha1.ConfigHookActionExecutableSpec, configFilePath string) error {
	actionTimeout, err := parseTimeout(action.Timeout)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, actionTimeout)
	defer cancel()

	dirExists, err := dirExists(action.Executable.WorkDir)
	if err != nil {
		return err
	}

	// we expect the directory to exist should be created by config if its new.
	if !dirExists {
		return os.ErrNotExist
	}

	// replace file token in args if it exists
	tokenMap := newTokenMap(configFilePath)
	args, err := replaceTokensInArgs(action.Executable.Args, tokenMap)
	if err != nil {
		return err
	}
	_, stderr, exitCode := exec.ExecuteWithContextFromDir(ctx, action.Executable.WorkDir, action.Executable.Command, args...)
	if exitCode != 0 {
		return fmt.Errorf("failed to execute command: %s %d: %s", action.Executable.Command, exitCode, stderr)
	}

	return nil
}

// addOrReplaceHookHandler adds or replaces a hook handler in the manager. this function assumes a lock is held.
func addOrReplaceHookHandler(watcher Watcher, newHook *v1alpha1.DeviceConfigHookSpec, existingHandlers map[string]HookHandler) error {
	// build lookup for file operations
	opActions := make(map[fsnotify.Op][]v1alpha1.ConfigHookAction)
	for _, action := range newHook.Actions {
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

	newWatchPath := newHook.WatchPath
	// TODO: this is a fair amount of work to do on every update, we should consider optimizing this.
	existingHandlers[newHook.WatchPath] = HookHandler{
		DeviceConfigHookSpec: newHook,
		opActions:            opActions,
	}

	// watcher will error if the path is already being watched
	for _, existingWatchPath := range watcher.List() {
		if existingWatchPath == newWatchPath {
			return nil
		}
	}

	if err := watcher.Add(newWatchPath); err != nil {
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
	for _, watchPath := range m.watcher.List() {
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
	return false, fmt.Errorf("failed to check if directory exists: %w", err)
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
