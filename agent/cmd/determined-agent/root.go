package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type opts struct {
	logLevel string
	noColor  bool
}

var version = "dev"

func newRootCmd() *cobra.Command {
	o := opts{}

	cmd := &cobra.Command{
		Use:     "determined-agent",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := bindEnv("DET_", cmd); err != nil {
				return err
			}
			level, err := log.ParseLevel(o.logLevel)
			if err != nil {
				return err
			}
			log.SetLevel(level)
			if level == log.TraceLevel {
				log.SetReportCaller(true)
			}
			log.SetFormatter(&log.TextFormatter{
				FullTimestamp: true,
				ForceColors:   true,
				DisableColors: o.noColor,
			})
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&o.logLevel, "log-level", "l", "info",
		"set the logging level (can be one of: debug, info, warn, error, or fatal)")
	cmd.PersistentFlags().BoolVar(&o.noColor, "no-color", false, "disable colored output")

	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newRunCmd())

	return cmd
}
