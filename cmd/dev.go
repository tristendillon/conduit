package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/tristendillon/conduit/core/generator"
	"github.com/tristendillon/conduit/core/logger"
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

		generator := generator.NewRouteGenerator(wd)
		excludePaths := generator.Walker.Exclude

		fw, err := watcher.NewFileWatcher(wd, excludePaths)
		if err != nil {
			return fmt.Errorf("failed to create file watcher: %w", err)
		}
		fw.FileWatcher.AddOnStartFunc(func() error {
			logger.Info("File watcher started, watching directory: %s", wd)
			logger.Info("Press Ctrl+C to stop...")

			return generator.GenerateRouteTree(logger.DEBUG)
		})
		fw.FileWatcher.AddOnChangeFunc(func() error {
			startTime := time.Now()
			logger.Info("File changes detected, regenerating...")
			err := generator.GenerateRouteTree(logger.DEBUG)
			if err != nil {
				logger.Error("Failed to generate route tree: %v", err)
				return err
			}
			logger.Info("Route tree generated successfully in %dms", time.Since(startTime).Milliseconds())
			return nil
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
