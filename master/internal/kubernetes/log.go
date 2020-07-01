package kubernetes

import (
	"io"
	"reflect"
	"time"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/agent"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"

	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/pkg/actor"
)

type (
	streamLogs struct{}
)

type podLogStreamer struct {
	podInterface typedV1.PodInterface
	podName      string
	podHandler   *actor.Ref

	ctx *actor.Context
}

func newPodLogStreamer(
	podInterface typedV1.PodInterface,
	podName string,
	podHandler *actor.Ref,
) *podLogStreamer {
	return &podLogStreamer{
		podInterface: podInterface,
		podName:      podName,
		podHandler:   podHandler,
	}
}

func (p *podLogStreamer) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:

	case streamLogs:
		if err := p.receiveStreamLogs(ctx); err != nil {
			return err
		}

	default:
		ctx.Log().Error("unexpected message: ", reflect.TypeOf(msg))
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

// Write implements the io.Writer interface.
func (p *podLogStreamer) Write(log []byte) (n int, err error) {
	p.ctx.Tell(p.podHandler, sproto.ContainerLog{
		Timestamp: time.Now(),
		RunMessage: &agent.RunMessage{
			Value:   string(log),
			StdType: stdcopy.Stdout,
		},
	})
	return len(log), nil
}

func (p *podLogStreamer) receiveStreamLogs(ctx *actor.Context) error {
	logs := p.podInterface.GetLogs(p.podName, &v1.PodLogOptions{
		Follow: true,
		// TODO: switch over to using k8 timestamps.
		Timestamps: false,
	})

	logReader, err := logs.Stream()
	if err != nil {
		return errors.Wrapf(err, "failed to initialize log stream for pod: %s", p.podName)
	}

	p.ctx = ctx
	ctx.Log().Debugf("starting log streaming for pod %s", p.podName)
	for {
		_, err := io.Copy(p, logReader)
		if err != nil {
			ctx.Log().Debugf("error reading logs: ", err)
			break
		}
	}

	ctx.Self().Stop()
	return nil
}
