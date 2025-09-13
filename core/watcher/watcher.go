package watcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tristendillon/conduit/core/cache"
	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/models"
)

type FileWatcher interface {
	Watch() error
	debounceGenerate()
	Close() error
	shouldExcludePath(path string) bool
	addWatchersRecursively(root string) error
	loadExcludePaths() error
}

type FileWatcherImpl struct {
	FileWatcher *models.FileWatcher
}

func NewFileWatcher(rootDir string, excludePaths []string) (*FileWatcherImpl, error) {
	fw, err := models.NewFileWatcher(rootDir, excludePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}
	return &FileWatcherImpl{
		FileWatcher: fw,
	}, nil
}

func (fw *FileWatcherImpl) Watch() error {
	if err := fw.addWatchersRecursively(fw.FileWatcher.RootDir); err != nil {
		return fmt.Errorf("failed to add watchers: %w", err)
	}

	if err := fw.FileWatcher.OnStart(); err != nil {
		logger.Error("Watcher.OnStart failed: %v", err)
	}

	for {
		select {
		case event, ok := <-fw.FileWatcher.Watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}

			if fw.shouldExcludePath(event.Name) {
				continue
			}

			logger.Debug("File event: %s %s", event.Op, event.Name)

			if strings.HasSuffix(event.Name, "route.go") {
				fileCache := cache.GetCache()
				if event.Has(fsnotify.Write) || event.Has(fsnotify.Remove) {
					fileCache.InvalidateFile(event.Name)
					logger.Debug("Invalidated cache for route file: %s", event.Name)
				}
			}

			if event.Has(fsnotify.Create) {
				if stat, err := os.Stat(event.Name); err == nil && stat.IsDir() {
					if !fw.shouldExcludePath(event.Name) {
						logger.Debug("Adding watcher for new directory: %s", event.Name)
						fw.FileWatcher.Watcher.Add(event.Name)
					}
				}
			}

			fw.debounceGenerate()

		case err, ok := <-fw.FileWatcher.Watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed")
			}
			logger.Error("Watcher error: %v", err)
		}
	}
}

func (fw *FileWatcherImpl) debounceGenerate() {
	fw.FileWatcher.Mutex.Lock()
	defer fw.FileWatcher.Mutex.Unlock()

	if fw.FileWatcher.DebounceTimer != nil {
		fw.FileWatcher.DebounceTimer.Stop()
	}

	fw.FileWatcher.DebounceTimer = time.AfterFunc(500*time.Millisecond, func() {
		logger.Debug("File changes detected, regenerating...")
		if err := fw.FileWatcher.OnChange(); err != nil {
			logger.Error("Watcher.OnChange failed: %v", err)
		}
	})
}

func (fw *FileWatcherImpl) Close() error {
	fw.FileWatcher.Mutex.Lock()
	defer fw.FileWatcher.Mutex.Unlock()

	if fw.FileWatcher.DebounceTimer != nil {
		fw.FileWatcher.DebounceTimer.Stop()
	}

	if err := fw.FileWatcher.OnClose(); err != nil {
		logger.Error("Watcher.OnClose failed: %v", err)
	}

	return fw.FileWatcher.Watcher.Close()
}

func (fw *FileWatcherImpl) shouldExcludePath(path string) bool {
	relPath, err := filepath.Rel(fw.FileWatcher.RootDir, path)
	if err != nil {
		return false
	}

	relPath = filepath.Clean(relPath)

	for _, excludePath := range fw.FileWatcher.ExcludePaths {
		excludePath = filepath.Clean(excludePath)

		if relPath == excludePath {
			return true
		}
		if strings.HasPrefix(relPath, excludePath+string(filepath.Separator)) {
			return true
		}
	}

	return false
}

func (fw *FileWatcherImpl) addWatchersRecursively(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			return nil
		}

		if fw.shouldExcludePath(path) {
			logger.Debug("Excluding directory: %s", path)
			return filepath.SkipDir
		}

		logger.Debug("Adding watcher for: %s", path)
		if err := fw.FileWatcher.Watcher.Add(path); err != nil {
			return fmt.Errorf("failed to add watcher for %s: %w", path, err)
		}

		return nil
	})
}
