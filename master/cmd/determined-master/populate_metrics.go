package main

import (
	"fmt"
	"os"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/internal/webhooks"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newPopulateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "populate",
		Short: "populate metrics",
		Run: func(cmd *cobra.Command, args []string) {
			if err := runPopulate(cmd, args); err != nil {
				log.Error(fmt.Sprintf("%+v", err))
				os.Exit(1)
			}
		},
	}
}

type apiServer struct {
	m *internal.Master

	usergroup.UserGroupAPIServer
	rbac.RBACAPIServerWrapper
	trials.TrialsAPIServer
	webhooks.WebhooksAPIServer
}

func runPopulate(cmd *cobra.Command, args []string) error {
	err := initializeConfig()
	if err != nil {
		return err
	}

	masterConfig := config.GetMasterConfig()
	database, err := db.Connect(&masterConfig.DB)
	if err != nil {
		return err
	}
	defer func() {
		if errd := database.Close(); errd != nil {
			log.Errorf("error closing pg connection: %s", errd)
		}
	}()

	return internal.PopulateExpTrialsMetrics(database, masterConfig)
}
