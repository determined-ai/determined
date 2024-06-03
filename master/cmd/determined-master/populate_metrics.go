package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/determined-ai/determined/master/internal"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
)

func newPopulateCmd() *cobra.Command {
	return &cobra.Command{
		Use: "populate NUM_BATCHES [ trivial ]",
		Short: `populate metrics with given number of batches. 
		trivial is an optional arg for trivial metrics rather than more complex ones.`,
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
	if len(args) < 1 {
		return errors.New("number of batches needs to be provided as the first argument")
	}
	batches, err := strconv.Atoi(args[0])
	if err != nil {
		return err
	}
	trivial := "trivial"
	isTrivial := false
	if len(args) >= 2 && args[1] == trivial {
		isTrivial = true
	} else if len(args) >= 2 && args[1] != trivial {
		return errors.New(`the second argument is optional. It should be the string: trivial which 
		indicates the function to populate the db with trivial metrics rather than more complex metrics`)
	}
	err = internal.PopulateExpTrialsMetrics(database, masterConfig, isTrivial, batches)
	fmt.Println("total time", time.Since(start)) //nolint:forbidigo
	return err
}
