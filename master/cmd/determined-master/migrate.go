package main

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/logger"
)

func newMigrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "migrate the db",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runMigrate(cmd, args); err != nil {
				log.Error(fmt.Sprintf("%+v", err))
				os.Exit(1)
			}
		},
	}
}

func runMigrate(cmd *cobra.Command, args []string) error {
	logStore := logger.NewLogBuffer(logStoreSize)
	log.AddHook(logStore)

	err := initializeConfig()
	if err != nil {
		return err
	}

	config := config.GetMasterConfig()
	database, err := db.Connect(&config.DB)
	if err != nil {
		return err
	}
	defer func() {
		if errd := database.Close(); errd != nil {
			log.Errorf("error closing pg connection: %s", errd)
		}
	}()

	if _, err = database.Migrate(config.DB.Migrations, args); err != nil {
		return errors.Wrap(err, "running migrations")
	}

	return nil
}
