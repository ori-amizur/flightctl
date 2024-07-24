package action

import (
	"github.com/fsnotify/fsnotify"
)

type Op uint32

const (
	Create Op = 1 << iota
	Write
	Remove
	Rename
	Chmod
)

func (op Op) String() string {
	var result string

	if op.Has(Create) {
		result += "|CREATE"
	}
	if op.Has(Remove) {
		result += "|REMOVE"
	}
	if op.Has(Write) {
		result += "|WRITE"
	}
	if op.Has(Rename) {
		result += "|RENAME"
	}
	if op.Has(Chmod) {
		result += "|CHMOD"
	}

	if result == "" {
		return "[no events]"
	}

	return result[1:]
}

func (o Op) Has(h Op) bool { return o&h == h }

type Handler struct {
	EventCallbackFn func(fsnotify.Event) error
}
