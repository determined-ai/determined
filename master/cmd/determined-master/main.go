package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/logger"
)

func main() {
	logger.SetLogrus(*logger.DefaultConfig())

	sigusr1 := make(chan os.Signal)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	exit := make(chan bool)
	go func() {
		if err := rootCmd.Execute(); err != nil {
			log.WithError(err).Fatal("fatal error running Determined master")
		}
		exit <- true
	}()

	select {
	case <-exit:
		return
	case <-sigusr1:
		log.Info("Got a SIGUSR1 quiting gracefully")
		os.Exit(198)
	}
}
