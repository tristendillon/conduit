/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "conduit",
	Short: "A Cli tool for building API services with Go.",
	Long: `Conduit is the go tool for connecting your go APIs with your frontend.
Utilizing Codegen to create solid RPC for your frontend and other services.
The REST version of gRPC.`,
}

var logfile string
var verbose bool

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logfile, "logfile", "", "File to write logs to")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output")
}
