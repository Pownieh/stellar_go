package main

import (
	"github.com/pownieh/stellar_go/services/regulated-assets-approval-server/cmd"
	"github.com/pownieh/stellar_go/support/log"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func main() {
	log.DefaultLogger = log.New()
	log.DefaultLogger.SetLevel(logrus.TraceLevel)

	rootCmd := &cobra.Command{
		Use:   "regulated-assets-approval-server [command]",
		Short: "SEP-8 Approval Server",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.AddCommand((&cmd.MigrateCommand{}).Command())
	rootCmd.AddCommand((&cmd.ServeCommand{}).Command())
	rootCmd.AddCommand((&cmd.ConfigureIssuer{}).Command())

	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
