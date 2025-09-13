package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/walker"
	"github.com/tristendillon/conduit/core/watcher"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run the dev command",
	Long:  "Looks for a main.go file in the current directory and reports its status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.SetVerbose(verbose)
		logger.Debug("dev called")
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		logger.Debug("Working directory: %s", wd)
		generateFunc := func(wd string) error {
			walker := walker.NewRouteWalker()
			if _, err := walker.Walk(wd); err != nil {
				return fmt.Errorf("failed to walk directory: %w", err)
			}

			walker.RouteTree.PrintTree(logger.DEBUG)
			return nil
		}

		fw, err := watcher.NewFileWatcher(wd)
		if err != nil {
			return fmt.Errorf("failed to create file watcher: %w", err)
		}
		fw.FileWatcher.AddOnStartFunc(func() error {
			logger.Info("File watcher started, watching directory: %s", wd)
			logger.Info("Press Ctrl+C to stop...")
			return generateFunc(wd)
		})
		fw.FileWatcher.AddOnChangeFunc(func() error {
			logger.Info("File changes detected, regenerating...")
			return generateFunc(wd)
		})
		fw.FileWatcher.AddOnCloseFunc(func() error {
			logger.Info("File watcher closed")
			return nil
		})
		if err := fw.Watch(); err != nil {
			return fmt.Errorf("failed to watch directory: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
}
