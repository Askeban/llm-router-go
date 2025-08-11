package config

import (
    "log"
    "path/filepath"
    "time"

    "github.com/fsnotify/fsnotify"
)

// WatchConfig watches the given file for changes and reloads the models into
// the provided store when modifications occur.  It runs a goroutine and
// returns immediately.  Any errors encountered by the watcher are logged
// through the provided logger.  A simple debounce prevents rapid successive
// reloads when multiple change events fire.
func WatchConfig(path string, store *ConfigStore, logger *log.Logger) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    // Determine directory and filename to watch
    absPath, err := filepath.Abs(path)
    if err != nil {
        return err
    }
    dir := filepath.Dir(absPath)
    file := filepath.Base(absPath)
    // Start goroutine for events
    go func() {
        defer watcher.Close()
        var lastReload time.Time
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }
                // Only act on events for our file
                if filepath.Base(event.Name) != file {
                    continue
                }
                if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
                    now := time.Now()
                    // Debounce: ignore events arriving within 200ms of the last reload
                    if now.Sub(lastReload) < 200*time.Millisecond {
                        continue
                    }
                    lastReload = now
                    models, err := LoadModels(path)
                    if err != nil {
                        logger.Printf("failed to reload models: %v", err)
                        continue
                    }
                    store.SetModels(models)
                    logger.Printf("reloaded %d models from %s", len(models), path)
                }
            case err, ok := <-watcher.Errors:
                if !ok {
                    return
                }
                logger.Printf("fsnotify error: %v", err)
            }
        }
    }()
    // Add directory to watcher
    if err := watcher.Add(dir); err != nil {
        return err
    }
    return nil
}