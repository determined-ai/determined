package kubernetes

import (
	"io"
	"time"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/pkg/errors"

	k8sV1 "k8s.io/api/core/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/agent"
	"github.com/determined-ai/determined/master/pkg/model"
)

type (
	streamLogs struct{}
)

type podLogStreamer struct {
	podHandler *actor.Ref
	logReader  io.ReadCloser

	ctx *actor.Context
}

func newPodLogStreamer(
	podInterface typedV1.PodInterface,
	podName string,
	podHandler *actor.Ref,
) (*podLogStreamer, error) {
	logs := podInterface.GetLogs(podName, &k8sV1.PodLogOptions{
		Follow:     true,
		Timestamps: false,
		Container:  model.DeterminedK8ContainerName,
	})

	logReader, err := logs.Stream()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize log stream for pod: %s", podName)
	}

	return &podLogStreamer{logReader: logReader, podHandler: podHandler}, nil
}

func (p *podLogStreamer) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		ctx.Tell(ctx.Self(), streamLogs{})

	case streamLogs:
		p.receiveStreamLogs(ctx)

	case actor.PostStop:

	default:
		ctx.Log().Errorf("unexpected message: %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

// Write implements the io.Writer interface.
func (p *podLogStreamer) Write(log []byte) (n int, err error) {
	p.ctx.Tell(p.podHandler, sproto.ContainerLog{
		Timestamp: time.Now().UTC(),
		RunMessage: &agent.RunMessage{
			Value:   string(log),
			StdType: stdcopy.Stdout,
		},
	})
	return len(log), nil
}

func (p *podLogStreamer) receiveStreamLogs(ctx *actor.Context) {
	p.ctx = ctx
	_, err := io.Copy(p, p.logReader)
	if err != nil {
		ctx.Log().WithError(err).Debug("error reading logs")
		ctx.Self().Stop()
		return
	}
	ctx.Tell(ctx.Self(), streamLogs{})
}
