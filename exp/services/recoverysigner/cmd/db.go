package cmd

import (
	"go/types"
	"strconv"
	"strings"

	dbpkg "github.com/pownieh/stellar_go/exp/services/recoverysigner/internal/db"
	"github.com/pownieh/stellar_go/exp/services/recoverysigner/internal/db/dbmigrate"
	"github.com/pownieh/stellar_go/support/config"
	supportlog "github.com/pownieh/stellar_go/support/log"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/spf13/cobra"
)

type DBCommand struct {
	Logger      *supportlog.Entry
	DatabaseURL string
}

func (c *DBCommand) Command() *cobra.Command {
	configOpts := config.ConfigOptions{
		{
			Name:        "db-url",
			Usage:       "Database URL",
			OptType:     types.String,
			ConfigKey:   &c.DatabaseURL,
			FlagDefault: "postgres://localhost:5432/?sslmode=disable",
			Required:    true,
		},
	}
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Run database operations",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			configOpts.Require()
			configOpts.SetValues()
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}
	configOpts.Init(cmd)

	migrateCmd := &cobra.Command{
		Use:   "migrate [up|down] [count]",
		Short: "Run migrations on the database",
		Run: func(cmd *cobra.Command, args []string) {
			c.Migrate(cmd, args)
		},
	}
	cmd.AddCommand(migrateCmd)

	return cmd
}

func (c *DBCommand) Migrate(cmd *cobra.Command, args []string) {
	db, err := dbpkg.Open(c.DatabaseURL)
	if err != nil {
		c.Logger.Errorf("Error opening database: %s", err.Error())
		return
	}

	if len(args) < 1 {
		cmd.Help()
		return
	}
	dirStr := args[0]

	var dir migrate.MigrationDirection
	switch dirStr {
	case "down":
		dir = migrate.Down
	case "up":
		dir = migrate.Up
	default:
		c.Logger.Errorf("Invalid migration direction, must be 'up' or 'down'.")
		return
	}

	var count int
	if len(args) >= 2 {
		count, err = strconv.Atoi(args[1])
		if err != nil {
			c.Logger.Errorf("Invalid migration count, must be a number.")
			return
		}
		if count < 1 {
			c.Logger.Errorf("Invalid migration count, must be a number greater than zero.")
			return
		}
	}

	migrations, err := dbmigrate.PlanMigration(db, dir, count)
	if err != nil {
		c.Logger.Errorf("Error planning migration: %s", err.Error())
		return
	}
	if len(migrations) > 0 {
		c.Logger.Infof("Migrations to apply %s: %s", dirStr, strings.Join(migrations, ", "))
	}

	n, err := dbmigrate.Migrate(db, dir, count)
	if err != nil {
		c.Logger.Errorf("Error applying migrations: %s", err.Error())
		return
	}
	if n > 0 {
		c.Logger.Infof("Successfully applied %d migrations %s.", n, dirStr)
	} else {
		c.Logger.Infof("No migrations applied %s.", dirStr)
	}
}
