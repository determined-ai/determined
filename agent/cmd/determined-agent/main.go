package main

import (
	"os"

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

	if err := newRootCmd().Execute(); err != nil {
		log.WithError(err).Fatal("fatal error running Determined agent")
	}
}
