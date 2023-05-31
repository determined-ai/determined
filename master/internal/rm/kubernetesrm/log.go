package kubernetesrm

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"

	"github.com/pkg/errors"

	k8sV1 "k8s.io/api/core/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/pkg/model"
)

type podLogStreamer struct {
	callback func(log []byte)
}

func startPodLogStreamer(
	podInterface typedV1.PodInterface,
	podName string,
	callback func(log []byte),
) error {
	logs := podInterface.GetLogs(podName, &k8sV1.PodLogOptions{
		Follow:     true,
		Timestamps: false,
		Container:  model.DeterminedK8ContainerName,
	})
	logReader, err := logs.Stream(context.TODO())
	if err != nil {
		return errors.Wrapf(err, "failed to initialize log stream for pod: %s", podName)
	}
	syslog := logrus.WithField("podName", podName)
	logger := &podLogStreamer{callback}

	go logger.receiveStreamLogs(syslog, logReader)

	return nil
}

// Write implements the io.Writer interface.
func (p *podLogStreamer) Write(log []byte) (n int, err error) {
	p.callback(log)
	return len(log), nil
}

func (p *podLogStreamer) receiveStreamLogs(
	syslog *logrus.Entry,
	logReader io.ReadCloser,
) {
	_, err := io.Copy(p, logReader)
	if err != nil {
		syslog.WithError(err).Debug("error reading logs")
	}
}
