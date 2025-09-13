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

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generates the routing tree for the project",
	Long:  `Generates the routing tree for the project`,
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
