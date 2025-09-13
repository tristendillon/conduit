package models

import (
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/tristendillon/conduit/core/config"
	"github.com/tristendillon/conduit/core/logger"
)

type FileWatcher struct {
	Watcher       *fsnotify.Watcher
	RootDir       string
	ExcludePaths  []string
	DebounceTimer *time.Timer
	Mutex         sync.Mutex
	OnStart       func() error
	OnChange      func() error
	OnClose       func() error
}

func NewFileWatcher(rootDir string, excludePaths []string) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	fw := &FileWatcher{
		Watcher:      watcher,
		RootDir:      rootDir,
		OnStart:      func() error { return fmt.Errorf("OnStart not set") },
		OnChange:     func() error { return fmt.Errorf("OnChange not set") },
		OnClose:      func() error { return fmt.Errorf("OnClose not set") },
		ExcludePaths: excludePaths,
	}

	if err := fw.loadExcludePaths(); err != nil {
		logger.Debug("Failed to load exclude paths from config: %v", err)
	}

	return fw, nil
}

func (fw *FileWatcher) AddOnStartFunc(onStart func() error) {
	fw.OnStart = onStart
}

func (fw *FileWatcher) AddOnChangeFunc(generateFunc func() error) {
	fw.OnChange = generateFunc
}

func (fw *FileWatcher) AddOnCloseFunc(onClose func() error) {
	fw.OnClose = onClose
}

func (fw *FileWatcher) loadExcludePaths() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fw.ExcludePaths = append(fw.ExcludePaths, []string{".git"}...)

	if cfg.Codegen.Go.Output != "" {
		fw.ExcludePaths = append(fw.ExcludePaths, cfg.Codegen.Go.Output)
	}
	if cfg.Codegen.Typescript.Output != "" {
		fw.ExcludePaths = append(fw.ExcludePaths, cfg.Codegen.Typescript.Output)
	}

	logger.Debug("Excluding paths: %v", fw.ExcludePaths)
	return nil
}
