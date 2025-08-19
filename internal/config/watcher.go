package config

import (
    "log"
    "path/filepath"
    
    "github.com/fsnotify/fsnotify"
)

func Watch(configPath string, onChange func(*Config)) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }

    go func() {
        defer watcher.Close()
        
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }
                
                if event.Op&fsnotify.Write == fsnotify.Write {
                    log.Println("Config file modified, reloading...")
                    
                    newConfig, err := Load(configPath)
                    if err != nil {
                        log.Printf("Failed to reload config: %v", err)
                        continue
                    }
                    
                    onChange(newConfig)
                }
                
            case err, ok := <-watcher.Errors:
                if !ok {
                    return
                }
                log.Printf("Config watcher error: %v", err)
            }
        }
    }()

    // Watch the directory, not just the file
    dir := filepath.Dir(configPath)
    return watcher.Add(dir)
}