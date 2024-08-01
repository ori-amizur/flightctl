package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/fsnotify/fsnotify"
)

type HookActionType string

const (
	SystemdHookActionType    = "Systemd"
	ExecutableHookActionType = "Executable"

	// FilePathKey is a placeholder which will be replaced with the file path
	FilePathKey = "FilePath"
)

var (
	ErrHookManagerNotInitialized = errors.New("hook manager not initialized")
	ErrInvalidTokenFormat        = errors.New("invalid token: format")
	ErrTokenNotSupported         = errors.New("invalid token: not supported")
)

type HookManager interface {
	Run(ctx context.Context)
	Update(hook *v1alpha1.DeviceConfigHookSpec) (bool, error)
	EnsurePostHooks([]v1alpha1.DeviceConfigHookSpec) error
	WatchList() []string
	HandleErrors() []error
	ResetDefaults() error
}

type FileMonitor interface {
	WatchAdd(name string) error
	WatchRemove(name string) error
	WatchList() []string
	Events() chan fsnotify.Event // TODO: hide implementation details
	Errors() chan error
	Close() error
}

type HookHandler struct {
	mu sync.Mutex
	*v1alpha1.DeviceConfigHookSpec
	opActions map[fsnotify.Op][]v1alpha1.ConfigHookAction
	err       error
}

func (h *HookHandler) Actions(op fsnotify.Op) ([]v1alpha1.ConfigHookAction, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	actions, ok := h.opActions[op]
	return actions, ok
}

func (h *HookHandler) Error() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.err
}

func (h *HookHandler) SetError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.err = err
}

func fileOperationToFsnotifyOp(op v1alpha1.FileOperation) (fsnotify.Op, error) {
	switch op {
	case v1alpha1.FileOperationCreate:
		return fsnotify.Create, nil
	case v1alpha1.FileOperationUpdate:
		return fsnotify.Write, nil
	case v1alpha1.FileOperationDelete:
		return fsnotify.Remove, nil
	default:
		return 0, fmt.Errorf("unsupported file operation: %s", op)
	}
}

// replaceTokensInArgs replaces tokens in the args with values from the token
// map. This should be more efficient than using go templates for this use-case
func replaceTokensInArgs(args []string, tokenMap map[string]string) ([]string, error) {
	var result []string
	for _, arg := range args {
		var out string
		index := 0
		for {
			start := strings.Index(arg[index:], "{{")
			if start == -1 {
				out = out + arg[index:]
				break
			} else {
				end := strings.Index(arg[index+start:], "}}")
				if end == -1 {
					out = out + arg[index:]
					break
				}
				trimmedToken := strings.TrimSpace(arg[index+start : index+start+end+2])
				parsedToken, err := parseToken(trimmedToken)
				if err != nil {
					return nil, err
				}
				if tokenData, ok := tokenMap[parsedToken]; ok {
					out = out + arg[index:index+start] + tokenData
					index = index + start + end + 2
				} else {
					return nil, fmt.Errorf("%w: %s", ErrTokenNotSupported, trimmedToken)
				}
			}
		}
		result = append(result, out)
	}
	return result, nil
}

func newTokenMap(filePath string) map[string]string {
	return map[string]string{
		FilePathKey: filePath,
	}
}

func parseToken(token string) (string, error) {
	parsed := strings.TrimSpace(token[2 : len(token)-2])
	if strings.HasPrefix(parsed, ".") {
		return parsed[1:], nil
	}
	return "", fmt.Errorf("%w: %s", ErrInvalidTokenFormat, token)
}
