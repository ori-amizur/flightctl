package config

import (
	"context"
	"errors"
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
)

type HookManager interface {
	Run(ctx context.Context)
	Update(hook *v1alpha1.DeviceConfigHookSpec) (bool, error)
	ResetDefaults() error
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
// map. This should be more efficient than using go templates for this use-case.
func replaceTokensInArgs(args []string, tokenMap map[string]string) []string {
	var result []string
	for _, arg := range args {
		trimmedArg := strings.TrimSpace(arg)
		if strings.HasPrefix(trimmedArg, "{{") && strings.HasSuffix(trimmedArg, "}}") {
			// remove the {{ and }} from the token and trim any spaces
			body := strings.TrimSpace(trimmedArg[2 : len(trimmedArg)-2])
			switch body {
			case "." + FilePathKey:
				if tokenData, ok := tokenMap[FilePathKey]; ok {
					result = append(result, tokenData)
				}
			default:
				result = append(result, arg)
			}
		} else {
			result = append(result, arg)
		}
	}

	return result
}

func newTokenMap(filePath string) map[string]string {
	return map[string]string{
		"FilePath": filePath,
	}
}
