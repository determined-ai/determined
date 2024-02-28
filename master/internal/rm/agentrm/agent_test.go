package agentrm

import (
	"errors"
	"testing"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/pkg/aproto"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
	"github.com/stretchr/testify/require"
)

func TestAgentFastFailAfterFirstConnect(t *testing.T) {
	a := newAgent(
		"test",
		queue.New[agentUpdatedEvent](),
		"default",
		&config.ResourcePoolConfig{},
		&aproto.MasterSetAgentOptions{},
		nil,
		func() {},
	)
	require.NotPanics(t, func() {
		a.stop(errors.New("agent immediately failed for some weird reason"))
	})
}
