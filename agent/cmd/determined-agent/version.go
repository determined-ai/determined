package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf( //nolint: forbidigo
				"Determined agent %s (built with %s)\n", version, runtime.Version(),
			)
		},
	}
}
