/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/walker"
)

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.SetVerbose(verbose)
		logger.Debug("generate called")
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
		logger.Debug("Working directory: %s", wd)
		walker := walker.NewRouteWalker()
		if _, err := walker.Walk(wd); err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		walker.RouteTree.PrintTree(logger.INFO)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
