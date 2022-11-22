package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/determined-ai/determined/master/pkg/logger"
)

type options struct {
	logger.Config

	logLevel string
	noColor  bool
}

var version = "dev"

func newRootCmd() *cobra.Command {
	opts := options{}

	cmd := &cobra.Command{
		Use:     "determined-agent",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := bindEnv("DET_", cmd); err != nil {
				return err
			}

			// Case hell for backwards compatibility - remove whenever, after 0.20 series.
			var usedDeprecatedLevel bool
			switch {
			case opts.logLevel != "" && opts.Level != "":
				return fmt.Errorf("cannot use `--log-level` and `--level`")
			case opts.logLevel != "" && opts.Level == "":
				usedDeprecatedLevel = true
				opts.Level = opts.logLevel
			case opts.logLevel == "" && opts.Level == "":
				opts.Level = "info"
			}

			var usedDeprecatedColor bool
			if opts.noColor && opts.Color {
				usedDeprecatedColor = true
				opts.Color = !opts.noColor
			}

			logger.SetLogrus(opts.Config)

			switch {
			case usedDeprecatedLevel:
				logrus.Warn("use of flag deprecated flag `--log-level`, please upgrade to `--level`")
			case usedDeprecatedColor:
				logrus.Warn("use of flag deprecated flag `--no-color`, please upgrade to `--color`")
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.logLevel, "log-level", "l", "",
		"set the logging level (can be one of: debug, info, warn, error, or fatal)")
	cmd.PersistentFlags().StringVar(&opts.Level, "level", "",
		"set the logging level (can be one of: debug, info, warn, error, or fatal)")
	cmd.PersistentFlags().BoolVar(&opts.noColor, "no-color", false, "disable colored output")
	cmd.PersistentFlags().BoolVar(&opts.Color, "color", true, "enable colored output")
	cmd.PersistentFlags().BoolVar(&opts.Structured, "structured", true, "enable structured logging")

	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newRunCmd())

	return cmd
}
