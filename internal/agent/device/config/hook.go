package config

import (
	"context"
	"errors"
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
	"github.com/samber/lo"
)

const (
	DefaultHookActionTimeout = 10 * time.Second
)

var _ HookManager = (*hookManager)(nil)

type hookManager struct {
	mu            sync.Mutex
	fsMonitor     FileMonitor
	handlers      map[string]*HookHandler
	systemdClient *client.Systemd
	exec          executer.Executer

	log *log.PrefixLogger
}

// NewHookManager creates a new device action manager.
func NewHookManager(log *log.PrefixLogger, exec executer.Executer) (HookManager, error) {
	fsMonitor, err := NewNotifyFileMonitor()
	if err != nil {
		return nil, err
	}

	return &hookManager{
		fsMonitor:     fsMonitor,
		systemdClient: client.NewSystemd(exec),
		exec:          exec,
		log:           log,
	}, nil
}

// Run starts the hook manager and listens for events.
func (m *hookManager) Run(ctx context.Context) {
	m.log.Info("Running hook manager")
	m.initialize()
	defer func() {
		if err := m.fsMonitor.Close(); err != nil {
			m.log.Errorf("Error closing watcher: %v", err)
		}
		m.log.Infof("Hook manager stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			m.log.Errorf("Context cancelled, shutting down")
			return
		case event, ok := <-m.fsMonitor.Events():
			if !ok {
				m.log.Debug("Watcher events channel closed")
				return
			}
			m.Handle(ctx, event)
		case err, ok := <-m.fsMonitor.Errors():
			if !ok {
				m.log.Debug("Watcher errors channel closed")
				return
			}
			m.log.Errorf("error: %v", err)
		}
	}

}
func (m *hookManager) Update(hook *v1alpha1.DeviceConfigHookSpec) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.update(hook)
}

// Update the manager with the new hook if appropriate.
func (m *hookManager) update(hook *v1alpha1.DeviceConfigHookSpec) (bool, error) {
	if m.handlers == nil {
		return false, ErrHookManagerNotInitialized
	}

	handler, ok := m.handlers[hook.WatchPath]
	if !ok || !reflect.DeepEqual(hook, handler.DeviceConfigHookSpec) {
		m.log.Infof("adding hook for watch path %s", hook.WatchPath)
		return true, addOrReplaceHookHandler(m.fsMonitor, hook, m.handlers)
	}

	return false, nil
}

func (m *hookManager) HandleErrors() []error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for _, handler := range m.handlers {
		if err := handler.Error(); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (m *hookManager) WatchList() []string {
	return m.fsMonitor.WatchList()
}

func (m *hookManager) watchRemove(watchPath string) error {
	if m.handlers == nil {
		return ErrHookManagerNotInitialized
	}

	if _, ok := m.handlers[watchPath]; ok {
		if err := m.fsMonitor.WatchRemove(watchPath); err != nil {
			return fmt.Errorf("failed removing watch: %w", err)
		}
		delete(m.handlers, watchPath)
	}
	return nil
}

func (m *hookManager) Handle(ctx context.Context, event fsnotify.Event) {
	filePath := event.Name
	m.log.Infof("Received event for %s", event.String())
	handle := m.getHandler(filePath)
	if handle == nil {
		m.log.Infof("no handler found for %s", event.String())
		// no handler for this event
		return
	}

	actions, ok := handle.Actions(event.Op)
	if !ok {
		// handle does not have any actions for this file operation
		m.log.Infof("no action for op %v", event.String())
		return
	}

	// actions
	var errs []error
	for i := range actions {
		action := actions[i]
		hookActionType, err := action.Discriminator()
		if err != nil {
			errs = append(errs, err)
			continue
		}

		switch hookActionType {
		case SystemdHookActionType:
			hookAction, err := action.AsConfigHookActionSystemdSpec()
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if err := handleHookActionSystemd(ctx, m.log, m.systemdClient, &hookAction, filePath); err != nil {
				errs = append(errs, err)
			}
		case ExecutableHookActionType:
			hookAction, err := action.AsConfigHookActionExecutableSpec()
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if err := handleHookActionExecutable(ctx, m.log, m.exec, &hookAction, filePath); err != nil {
				errs = append(errs, err)
			}
		default:
			errs = append(errs, fmt.Errorf("unknown hook action type: %s", hookActionType))
		}
	}

	// push errors to status
	if err := errors.Join(errs...); err != nil {
		m.log.Errorf("handle error: %+v", err)
	}
	handle.SetError(errors.Join(errs...))
}

func (m *hookManager) getHandler(eventName string) *HookHandler {
	m.mu.Lock()
	defer m.mu.Unlock()
	return getHandler(eventName, m.handlers)
}

func (m *hookManager) initialize() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// initialize the handlers map here for testing observability.
	m.handlers = make(map[string]*HookHandler)
}

func (m *hookManager) ResetDefaults() error {
	return m.EnsurePostHooks(nil)
}

func mergeHooks(h1, h2 []v1alpha1.DeviceConfigHookSpec) []v1alpha1.DeviceConfigHookSpec {
	hooksMap := make(map[string]v1alpha1.DeviceConfigHookSpec)
	for _, hook := range append(h1, h2...) {
		h, ok := hooksMap[hook.WatchPath]
		if ok {
			h.Actions = append(h.Actions, hook.Actions...)
		} else {
			hooksMap[hook.WatchPath] = hook
		}
	}
	return lo.Values(hooksMap)
}

func (m *hookManager) EnsurePostHooks(hooks []v1alpha1.DeviceConfigHookSpec) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newWatchPaths := make(map[string]struct{})
	for _, hook := range mergeHooks(hooks, defaultPostHooks()) {
		newWatchPaths[hook.WatchPath] = struct{}{}
		updated, err := m.update(&hook)
		if err != nil {
			return err
		}
		if updated {
			m.log.Infof("Updated hook: %s", hook.Name)
		}
	}

	existingWatchPaths := m.WatchList()
	for _, existingWatchPath := range existingWatchPaths {
		if _, ok := newWatchPaths[existingWatchPath]; !ok {
			if err := m.watchRemove(existingWatchPath); err != nil {
				return err
			}
			m.log.Infof("Removed watch: %s", existingWatchPath)
		}
	}

	return nil
}

func handleHookActionSystemd(ctx context.Context, log *log.PrefixLogger, systemdClient *client.Systemd, action *v1alpha1.ConfigHookActionSystemdSpec, filePath string) error {
	actionTimeout, err := parseTimeout(action.Timeout)
	if err != nil {
		return err
	}

	var unitName string
	if action.Unit.Name != "" {
		unitName = action.Unit.Name
	} else {
		// attempt to extract the systemd unit name from the file path
		unitName, err = getSystemdUnitNameFromFilePath(filePath)
		if err != nil {
			log.Errorf("%v: skipping...", err)
			return nil
		}
	}

	for _, op := range action.Unit.Operations {
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

func handleHookActionExecutable(ctx context.Context, log *log.PrefixLogger, exec executer.Executer, action *v1alpha1.ConfigHookActionExecutableSpec, configFilePath string) error {
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
	// TODO: cache the map.
	tokenMap := newTokenMap(configFilePath)
	args, err := replaceTokensInArgs(action.Executable.Args, tokenMap)
	if err != nil {
		return err
	}

	var envVars []string
	if action.Executable.EnvVars != nil {
		envVars = *action.Executable.EnvVars
	}
	log.Infof("executing %s with args %+v and environment variables %+v", action.Executable.Command, args, envVars)
	_, stderr, exitCode := exec.ExecuteWithContextFromDir(ctx, action.Executable.WorkDir, action.Executable.Command, args, envVars...)
	if exitCode != 0 {
		return fmt.Errorf("failed to execute command: %s %d: %s", action.Executable.Command, exitCode, stderr)
	}

	return nil
}

// addOrReplaceHookHandler adds or replaces a hook handler in the manager. this function assumes a lock is held.
func addOrReplaceHookHandler(fsMonitor FileMonitor, newHook *v1alpha1.DeviceConfigHookSpec, existingHandlers map[string]*HookHandler) error {
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
				notifyOp, err := fileOperationToFsnotifyOp(op)
				if err != nil {
					return err
				}

				opActions[notifyOp] = append(opActions[notifyOp], action)
			}
		case ExecutableHookActionType:
			configHook, err := action.AsConfigHookActionExecutableSpec()
			if err != nil {
				return err
			}
			if configHook.Executable.EnvVars != nil {
				if err := validateEnvVars(*configHook.Executable.EnvVars); err != nil {
					return err
				}
			}
			for _, op := range configHook.TriggerOn {
				notifyOp, err := fileOperationToFsnotifyOp(op)
				if err != nil {
					return err
				}
				opActions[notifyOp] = append(opActions[notifyOp], action)
			}
		default:
			return fmt.Errorf("unknown hook action type: %s", hookActionType)
		}
	}

	newWatchPath := newHook.WatchPath
	// TODO: this is a fair amount of work to do on every update, we should consider optimizing this.
	existingHandlers[newHook.WatchPath] = &HookHandler{
		DeviceConfigHookSpec: newHook,
		opActions:            opActions,
	}

	// watcher will error if the path is already being watched
	for _, existingWatchPath := range fsMonitor.WatchList() {
		if existingWatchPath == newWatchPath {
			return nil
		}
	}

	if err := fsMonitor.WatchAdd(newWatchPath); err != nil {
		return fmt.Errorf("failed adding watch: %w", err)
	}

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

func getHandler(eventName string, handlers map[string]*HookHandler) *HookHandler {
	// check for exact match on file
	if handler, exists := handlers[eventName]; exists {
		return handler
	}
	// fallback to dir
	parentDir := filepath.Dir(eventName)
	if handler, exists := handlers[parentDir]; exists {
		return handler
	}

	return nil
}

func validateEnvVars(envVars []string) error {
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid envVar format: should be KEY=VALUE: %s", envVar)
		}
		key, value := parts[0], parts[1]
		if key == "" {
			return fmt.Errorf("invalid envVar format: key cannot be empty: %s", envVar)
		}
		if strings.Contains(key, " ") {
			return fmt.Errorf("invalid envVar format: key cannot contain spaces: %s", envVar)
		}
		if value == "" {
			return fmt.Errorf("invalid envVar format: value cannot be empty: %s", envVar)
		}
		if key != strings.ToUpper(key) {
			return fmt.Errorf("invalid envVar format: key must be uppercase: %s", envVar)
		}
	}
	return nil
}

func marshalExecutable(command string, args []string, envVars *[]string, workDir string, timeout string, triggerOn []v1alpha1.FileOperation) v1alpha1.ConfigHookAction {
	cha := v1alpha1.ConfigHookActionExecutableSpec{
		Executable: v1alpha1.ConfigHookActionExecutable{
			Args:    args,
			EnvVars: envVars,
			Command: command,
			WorkDir: workDir,
		},
		Timeout:   &timeout,
		TriggerOn: triggerOn,
	}
	var ret v1alpha1.ConfigHookAction
	_ = ret.FromConfigHookActionExecutableSpec(cha)
	return ret
}

func defaultPostHooks() []v1alpha1.DeviceConfigHookSpec {
	return []v1alpha1.DeviceConfigHookSpec{
		{
			Name:        "compose-action-hook",
			WatchPath:   "/var/run/flightctl/compose",
			Description: lo.ToPtr(""),
			Actions: []v1alpha1.ConfigHookAction{
				marshalExecutable("docker-compose", []string{"-f", "{{ .FilePath }}", "up", "-d"}, lo.ToPtr([]string{"DOCKER_HOST=unix:///run/podman/podman.sock"}),
					"/var/run/flightctl/compose", "1m", []v1alpha1.FileOperation{v1alpha1.FileOperationCreate, v1alpha1.FileOperationUpdate}),
			},
		},
		{
			Name:        "quadlet-action-hook",
			WatchPath:   "/etc/containers/systemd",
			Description: lo.ToPtr(""),
			Actions: []v1alpha1.ConfigHookAction{
				marshalExecutable("bash", []string{"-c", `systemctl daemon-reload && fname=$(basename {{ .FilePath }}) && srv=$([[ "${fname##*.}" == "container" ]] && echo -n "${fname%.*}" || echo -n ${fname//./-}).service && systemctl start $srv`}, nil,
					"/etc/containers/systemd", "1m", []v1alpha1.FileOperation{v1alpha1.FileOperationCreate, v1alpha1.FileOperationUpdate}),
			},
		},
	}
}
