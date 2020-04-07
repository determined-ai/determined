package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/determined-ai/determined/agent/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Determined agent %s (built with %s)\n", version.Version, runtime.Version())
		},
	}
}
