// go:build dev
//go:build dev

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tristendillon/conduit/core/logger"
	"github.com/tristendillon/conduit/core/template_engine/template_refs"
)

var generateTemplateRefsCmd = &cobra.Command{
	Use:   "generate-template-refs <templates-path>",
	Short: "Dev command for generating template references for the project.",
	Long:  `Dev command for generating template references for the project.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.SetVerbose(verbose)
		logger.Debug("generate-template-refs called")
		templatesDir := args[0]
		walker := template_refs.NewTemplateWalker(templatesDir)
		if err := walker.Walk(); err != nil {
			return fmt.Errorf("failed to walk templates directory: %w", err)
		}

		files := walker.GetFileNodes()
		dirs := walker.GetDirectoryNodes()

		generator := template_refs.NewTemplateGenerator(walker)

		logger.Debug("Discovery Summary:")
		logger.Debug("   Files found: %d", len(files))
		logger.Debug("   Directories found: %d", len(dirs))
		logger.Debug("Template Structure:")
		generator.PrintTemplateTree()

		if err := generator.Generate(); err != nil {
			return fmt.Errorf("failed to generate template references: %w", err)
		}

		logger.Info("Successfully generated template references!")
		logger.Info("Generated references for %d files and %d directories", len(files), len(dirs))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateTemplateRefsCmd)
}
