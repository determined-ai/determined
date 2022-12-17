package internal

import (
	"context"
	"fmt"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/agent/internal/options"
	"github.com/determined-ai/determined/master/pkg/syncx/errgroupx"
)

// Run runs a new agent system and actor with the provided options.
func Run(parent context.Context, version string, opts options.AgentOptions) error {
	ctx, stop := signal.NotifyContext(parent, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	printableConfig, err := opts.Printable()
	if err != nil {
		return err
	}
	log.Infof("agent configuration: %s", printableConfig)

	wg := errgroupx.WithContext(ctx)

	log.Trace("starting main agent process")
	wg.Go(func(ctx context.Context) error {
		err := New(ctx, version, opts).Wait()
		if _, ok := err.(longDisconnected); ok {
			onConnectionLost(ctx, opts)
		}
		return err
	})

	if opts.APIEnabled {
		log.Trace("starting agent apiserver")
		wg.Go(func(ctx context.Context) error {
			if err := newAgentAPIServer(opts).serve(); err != nil {
				return fmt.Errorf("api server crashed: %w", err)
			}
			return errors.New("api server exited unexpectedly")
		})
	}

	return wg.Wait()
}

func onConnectionLost(ctx context.Context, opts options.AgentOptions) {
	cmd := opts.Hooks.OnConnectionLost
	if len(cmd) == 0 {
		return
	}
	out, err := exec.CommandContext(ctx, cmd[0], cmd[1:]...).CombinedOutput() //nolint:gosec
	if err != nil {
		log.
			WithError(err).
			WithField("output", string(out)).
			Error("error running connection failure hook")
	}
}
