package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
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

func runPopulate(cmd *cobra.Command, args []string) error {
	start := time.Now()
	err := initializeConfig()
	if err != nil {
		return err
	}

	masterConfig := config.GetMasterConfig()
	database, err := db.Setup(&masterConfig.DB)
	if err != nil {
		return err
	}

	if err = etc.SetRootPath(filepath.Join(masterConfig.Root, "static/srv")); err != nil {
		return err
	}

	defer func() {
		if errd := database.Close(); errd != nil {
			log.Errorf("error closing pg connection: %s", errd)
		}
	}()

	batches, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	trivial := false
	fmt.Println(args)
	if len(args) >= 2 && args[1] == "trivial" {
		trivial = true
	}

	err = internal.PopulateExpTrialsMetrics(database, masterConfig, trivial, batches)
	fmt.Println("total time", time.Since(start))
	return err
}
