/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/template_engine"
)

var (
	force bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new Conduit project",
	Long:  `Creates the boilplate and necessary files for a new Conduit project.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		logger.SetVerbose(verbose)
		logger.Debug("init called")
		dir := args[0]
		if _, err := os.Stat(dir); err == nil {
			if !force {
				fmt.Printf("Directory %s already exists. Use --force to overwrite.\n", dir)
				return
			} else {
				logger.Debug("Directory %s already exists. Overwriting.", dir)
				os.RemoveAll(dir)
			}
		}
		initData := map[string]string{
			"ModuleName": strings.ToLower(dir),
		}
		os.MkdirAll(dir, os.ModePerm)
		engine := template_engine.NewTemplateEngine()
		if err := engine.GenerateFolder(template_engine.TEMPLATES.INIT.Ref, dir, initData); err != nil {
			fmt.Printf("Failed to generate project: %v\n", err)
			return
		}
		fmt.Printf("Successfully generated project: %s\n", dir)

		failure := false
		if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
			fmt.Printf("Failed to install dependencies: %v\n", err)
			failure = true
		}

		fmt.Printf("Next Steps:\n")
		fmt.Printf("  - cd %s\n", dir)
		if failure {
			fmt.Printf("  - go mod tidy\n")
		}
		fmt.Printf("  - conduit dev\n")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().BoolVar(&force, "force", false, "Force overwrite existing files")
}
