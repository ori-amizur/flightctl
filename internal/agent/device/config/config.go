package config

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	WatchList() []string
	WatchRemove(name string) error
	ResetDefaults() error
}

type Watcher interface {
	Add(name string) error
	Remove(name string) error
	List() []string
	Events() chan fsnotify.Event // TODO: hide imlementation details
	Errors() chan error
	Close() error
}

type HookHandler struct {
	*v1alpha1.DeviceConfigHookSpec
	opActions map[fsnotify.Op][]v1alpha1.ConfigHookAction
}

func fileOperationToFsnotifyOp(op v1alpha1.FileOperation) fsnotify.Op {
	switch op {
	case v1alpha1.FileOperationCreate:
		return fsnotify.Create
	case v1alpha1.FileOperationUpdate:
		return fsnotify.Write
	case v1alpha1.FileOperationDelete:
		return fsnotify.Remove
	case v1alpha1.FileOperationRename:
		return fsnotify.Rename
	case v1alpha1.FileOperationChangePermissions:
		return fsnotify.Chmod
	default:
		return fsnotify.Op(0)
	}
}

// replaceTokensInArgs replaces tokens in the args with values from the token
// map. This should be more efficient than using go templates for this use-case
func replaceTokensInArgs(args []string, tokenMap map[string]string) ([]string, error) {
	var result []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "{{") && strings.HasSuffix(arg, "}}") {
			trimmedArg := strings.TrimSpace(arg)
			parsedToken, err := parseToken(trimmedArg)
			if err != nil {
				return nil, err
			}
			if tokenData, ok := tokenMap[parsedToken]; ok {
				result = append(result, tokenData)
			} else {
				return nil, fmt.Errorf("%w: %s", ErrTokenNotSupported, arg)
			}
		} else {
			result = append(result, arg)
		}
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
