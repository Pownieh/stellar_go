package cmd

import (
	horizon "github.com/pownieh/stellar_go/services/horizon/internal"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "run horizon server",
	Long:  "serve initializes then starts the horizon HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		app, err := horizon.NewAppFromFlags(globalConfig, globalFlags)
		if err != nil {
			return err
		}
		return app.Serve()
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)
}
