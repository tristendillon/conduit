package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// devCmd represents the dev command
var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Run the dev command",
	Long:  "Looks for a main.go file in the current directory and reports its status.",
	RunE: func(cmd *cobra.Command, args []string) error {
		wd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		mainPath := filepath.Join(wd, "main.go")
		if _, err := os.Stat(mainPath); os.IsNotExist(err) {
			return fmt.Errorf("main.go not found in %s", wd)
		} else if err != nil {
			return fmt.Errorf("error checking for main.go: %w", err)
		}

		fmt.Printf("✅ Found main.go at %s\n", mainPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(devCmd)
}
