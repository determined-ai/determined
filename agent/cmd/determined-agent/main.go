package main

import (
	log "github.com/sirupsen/logrus"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		log.WithError(err).Fatal("fatal error running Determined agent")
	}
}
