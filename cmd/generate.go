/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tristendillon/conduit/core/generator"
	"github.com/tristendillon/conduit/core/logger"
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

		generator := generator.NewRouteGenerator(wd)
		if err := generator.GenerateRouteTree(logger.INFO); err != nil {
			return fmt.Errorf("failed to generate route tree: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)
}
