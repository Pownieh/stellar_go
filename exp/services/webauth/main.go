package main

import (
	"github.com/pownieh/stellar_go/exp/services/webauth/cmd"
	supportlog "github.com/pownieh/stellar_go/support/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	logger := supportlog.New()
	logger.SetLevel(logrus.TraceLevel)

	rootCmd := &cobra.Command{
		Use:   "webauth [command]",
		Short: "SEP-10 Web Authentication Server",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand((&cmd.ServeCommand{Logger: logger}).Command())
	rootCmd.AddCommand((&cmd.GenJWKCommand{Logger: logger}).Command())

	err := rootCmd.Execute()
	if err != nil {
		logger.Fatal(err)
	}
}
