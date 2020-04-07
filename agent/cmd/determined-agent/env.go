package main

import (
	"strings"
	"syscall"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func bindEnv(prefix string, cmd *cobra.Command) error {
	var errMsgs []string
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		envName := prefix + strings.ReplaceAll(strings.ToUpper(flag.Name), "-", "_")
		if value, ok := syscall.Getenv(envName); ok {
			if err := flag.Value.Set(value); err != nil {
				err = errors.Wrapf(err, "failed to parse %s (%s)", envName, flag.Value.Type())
				errMsgs = append(errMsgs, err.Error())
			}
		}
	})
	if len(errMsgs) == 0 {
		return nil
	}
	msg := strings.Join(errMsgs, ";")
	return errors.New(msg)
}
