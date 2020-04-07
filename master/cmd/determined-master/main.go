package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/logger"
)

func main() {
	logger.SetLogrus(*logger.DefaultConfig())

	if err := rootCmd.Execute(); err != nil {
		log.WithError(err).Fatal("fatal error running Determined master")
	}
}
