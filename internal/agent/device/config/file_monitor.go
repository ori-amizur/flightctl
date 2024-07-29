package config

import (
	fsnotify "github.com/fsnotify/fsnotify"
)

var _ FileMonitor = (*notifyFileMonitor)(nil)

type notifyFileMonitor struct {
	watcher *fsnotify.Watcher
	events  chan fsnotify.Event
}

func NewNotifyFileMonitor() (*notifyFileMonitor, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &notifyFileMonitor{
		watcher: watcher,
		events:  watcher.Events,
	}, nil
}

func (w *notifyFileMonitor) WatchAdd(name string) error {
	return w.watcher.Add(name)
}

func (w *notifyFileMonitor) WatchRemove(name string) error {
	return w.watcher.Remove(name)
}

func (w *notifyFileMonitor) WatchList() []string {
	return w.watcher.WatchList()
}

func (w *notifyFileMonitor) Events() chan fsnotify.Event {
	return w.events
}

func (w *notifyFileMonitor) Errors() chan error {
	return w.watcher.Errors
}

func (w *notifyFileMonitor) Close() error {
	return w.watcher.Close()
}
