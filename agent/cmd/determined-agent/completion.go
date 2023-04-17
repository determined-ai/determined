package main

import (
	"fmt"
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
				if err := newRootCmd().GenBashCompletion(os.Stdout); err != nil {
					return fmt.Errorf("error generating agent command bash completion: %w", err)
				}
			case zshCompletion:
				if err := newRootCmd().GenZshCompletion(os.Stdout); err != nil {
					return fmt.Errorf("error generating agent command zsh completion: %w", err)
				}
			case powerShellCompletion:
				if err := newRootCmd().GenPowerShellCompletion(os.Stdout); err != nil {
					return fmt.Errorf("error generating agent command power shell completion: %w", err)
				}
			default:
				return errors.Errorf("unexpected shell provided: %s", shell)
			}
			return nil
		},
	}
}
