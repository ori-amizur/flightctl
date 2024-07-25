package config

import (
	"github.com/fsnotify/fsnotify"
)

type EventHandler struct {
	EventCallbackFn func(fsnotify.Event) error
}
