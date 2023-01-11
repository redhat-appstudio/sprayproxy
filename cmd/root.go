/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "sprayproxy",
	Short: "A reverse proxy to broadcast to multiple backends",
	Long: `A reverse proxy to broadcast requests to multiple backend servers.

sprayproxy server --backend <backend-server> --backend <another-backend>
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Placeholder for additional flags (persistent or local)
}
