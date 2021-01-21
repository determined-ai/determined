package main

import (
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/logger"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	logger.SetLogrus(*logger.DefaultConfig())

	if err := rootCmd.Execute(); err != nil {
		log.WithError(err).Fatal("fatal error running Determined master")
	}
}
