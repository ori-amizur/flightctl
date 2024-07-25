package config

import (
	"context"

	"github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/fsnotify/fsnotify"
)

type HookActionType string

const (
	SystemdHookActionType    = "Systemd"
	ExecutableHookActionType = "Executable"
)

type HookManager interface {
	Run(ctx context.Context)
	Update(hook *v1alpha1.DeviceConfigHook) (bool, error)
	ResetDefaults() error
}

type HookHandler struct {
	*v1alpha1.DeviceConfigHook
	FileOpLookup map[fsnotify.Op]struct{}
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
