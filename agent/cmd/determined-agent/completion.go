package main

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	bashCompletion       = "bash"
	zshCompletion        = "zsh"
	powerShellCompletion = "power"
)

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "completion",
		Short:     "generates shell completion scripts",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{bashCompletion, zshCompletion, powerShellCompletion},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch shell := args[0]; shell {
			case bashCompletion:
				return newRootCmd().GenBashCompletion(os.Stdout)
			case zshCompletion:
				return newRootCmd().GenZshCompletion(os.Stdout)
			case powerShellCompletion:
				return newRootCmd().GenPowerShellCompletion(os.Stdout)
			default:
				return errors.Errorf("unexpected shell provided: %s", shell)
			}
		},
	}
}
