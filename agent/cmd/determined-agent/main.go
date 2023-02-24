package main

import (
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func maybeInjectRootAlias(rootCmd *cobra.Command, inject string) {
	nonRootAliases := nonRootSubCmds(rootCmd)

	if len(os.Args) > 1 {
		for _, v := range nonRootAliases {
			if os.Args[1] == v {
				return
			}
		}
	}
	os.Args = append([]string{os.Args[0], inject}, os.Args[1:]...)
}

func nonRootSubCmds(rootCmd *cobra.Command) []string {
	res := []string{"help"}
	for _, c := range rootCmd.Commands() {
		res = append(res, c.Name())
		res = append(res, c.Aliases...)
	}

	return res
}

func main() {
	rootCmd := newRootCmd()
	maybeInjectRootAlias(rootCmd, "run")

	sigusr1 := make(chan os.Signal)
	signal.Notify(sigusr1, syscall.SIGUSR1)

	exit := make(chan bool)
	go func() {
		if err := newRootCmd().Execute(); err != nil {
			log.WithError(err).Fatal("fatal error running Determined agent")
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
