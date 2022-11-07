package internal

import (
	"context"
	"fmt"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/agent/internal/detect"
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

	log.Trace("detecting devices")
	devices, err := detect.Detect(opts.SlotType, opts.AgentID, opts.VisibleGPUs, opts.ArtificialSlots)
	if err != nil {
		return fmt.Errorf("failed to detect devices: %w", devices)
	}

	wg := errgroupx.WithContext(ctx)

	log.Trace("starting main agent process")
	wg.Go(func(ctx context.Context) error {
		connectionFailureWindowBegin := time.Now()
		connectionFailureCount := 0
		for {
			a := New(ctx, version, opts)
			switch err := a.Wait().(type) {
			case MasterConnectionError:
				now := time.Now()
				if connectionFailureWindowBegin.Before(now.Add(-time.Minute)) {
					connectionFailureWindowBegin = now
					connectionFailureCount = 0
				}
				connectionFailureCount++
				if connectionFailureCount >= opts.AgentReconnectAttempts {
					onConnectionLost(ctx, opts)
					return fmt.Errorf("failure to recover agent connection: %w", err)
				}
				log.WithError(err).Error("attempting reconnect after delay...")
				time.Sleep(time.Duration(opts.AgentReconnectBackoff))
				continue
			default:
				return err
			}
		}
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
