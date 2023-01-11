/*
Copyright © 2023 The Spray Proxy Contributors

SPDX-License-Identifier: Apache-2.0
*/
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/adambkaplan/sprayproxy/pkg/server"
)

// serverCmd represents the server command
var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the spray reverse proxy server",
	Long: `Run a reverse proxy that blindly "sprays" requests to one or more backend servers.
Use the --backend flag to specify which servers to forward traffic to:

sprayproxy server --backend http://localhost:8081 --backend http://localhost:8082
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		server, err := server.NewServer(host, port, backends...)
		if err != nil {
			return err
		}
		return server.Run()
	},
}

var (
	host     string
	port     int
	backends []string
)

func init() {
	rootCmd.AddCommand(serverCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serverCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serverCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	serverCmd.Flags().StringVar(&host, "host", "localhost", "Host for running the server. Defaults to localhost")
	serverCmd.Flags().IntVar(&port, "port", 8080, "Port for running the server. Defaults to 8080")
	serverCmd.Flags().StringSliceVar(&backends, "backend", []string{}, "Backend to forward requests. Use more than once.")

}
