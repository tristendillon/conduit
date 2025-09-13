/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tristendillon/conduit/core/version"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the version of Conduit",
	Long:  `Displays the version of Conduit.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Conduit %s\n", version.Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
