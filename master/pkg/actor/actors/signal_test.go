package actors

import (
	"syscall"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type signalListener struct{ t *testing.T }

func (s *signalListener) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case string:
		NotifyOnSignal(ctx, syscall.SIGWINCH, syscall.SIGCHLD)
	case syscall.Signal:
		if msg == syscall.SIGCHLD {
			ctx.Self().Stop()
		} else {
			assert.Equal(s.t, msg, syscall.SIGWINCH)
		}
	}
	return nil
}

func TestNotifyOnSignal(t *testing.T) {
	system := actor.NewSystem(t.Name())
	ref, _ := system.ActorOf(actor.Addr("test"), &signalListener{t})
	system.Ask(ref, "register").Get()
	assert.NilError(t, syscall.Kill(syscall.Getpid(), syscall.SIGPIPE))
	assert.NilError(t, syscall.Kill(syscall.Getpid(), syscall.SIGWINCH))
	assert.NilError(t, syscall.Kill(syscall.Getpid(), syscall.SIGCHLD))
	assert.NilError(t, ref.AwaitTermination())
}
