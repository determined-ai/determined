package kubernetesrm

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type requestProcessingWorker struct {
	podInterfaces       map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface
	syslog              *logrus.Entry
}

type readyCallbackFunc func(string)

func startRequestProcessingWorker(
	podInterfaces map[string]typedV1.PodInterface,
	configMapInterfaces map[string]typedV1.ConfigMapInterface,
	id string,
	in <-chan interface{},
	ready readyCallbackFunc,
) *requestProcessingWorker {
	syslog := logrus.New().WithField("component", "kubernetesrm-worker").WithField("id", id)
	r := &requestProcessingWorker{
		podInterfaces:       podInterfaces,
		configMapInterfaces: configMapInterfaces,
		syslog:              syslog,
	}
	go r.receive(in, ready)
	return r
}

func (r *requestProcessingWorker) receive(in <-chan interface{}, ready readyCallbackFunc) {
	go ready("")
	for msg := range in {
		switch msg := msg.(type) {
		case createKubernetesResources:
			r.receiveCreateKubernetesResources(msg)
			go ready(getKeyForCreate(msg))
		case deleteKubernetesResources:
			r.receiveDeleteKubernetesResources(msg)
			go ready("")
		default:
			errStr := fmt.Sprintf("unexpected message %T", msg)
			r.syslog.Error(errStr)
			panic(errStr)
		}
	}
}

func (r *requestProcessingWorker) receiveCreateKubernetesResources(
	msg createKubernetesResources,
) {
	r.syslog.Debugf("creating configMap with spec %v", msg.configMapSpec)
	configMap, err := r.configMapInterfaces[msg.podSpec.Namespace].Create(
		context.TODO(), msg.configMapSpec, metaV1.CreateOptions{})
	if err != nil {
		r.syslog.WithError(err).Errorf("error creating configMap %s", msg.configMapSpec.Name)
		go msg.errorHandler(resourceCreationFailed{err})
		return
	}
	r.syslog.Infof("created configMap %s", configMap.Name)

	r.syslog.Debugf("launching pod with spec %v", msg.podSpec)
	pod, err := r.podInterfaces[msg.podSpec.Namespace].Create(
		context.TODO(), msg.podSpec, metaV1.CreateOptions{},
	)
	if err != nil {
		r.syslog.WithError(err).Errorf("error creating pod %s", msg.podSpec.Name)
		go msg.errorHandler(resourceCreationFailed{err})
		return
	}
	r.syslog.Infof("created pod %s", pod.Name)
}

func (r *requestProcessingWorker) receiveDeleteKubernetesResources(
	msg deleteKubernetesResources,
) {
	var gracePeriod int64 = deletionGracePeriod
	var err error

	// If resource creation failed, we will still try to delete those resources which
	// will also result in a failure.
	if len(msg.podName) > 0 {
		err = r.podInterfaces[msg.namespace].Delete(
			context.TODO(), msg.podName, metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if err != nil {
			r.syslog.WithError(err).Errorf("failed to delete pod %s", msg.podName)
		} else {
			r.syslog.Infof("deleted pod %s", msg.podName)
		}
	}

	if len(msg.configMapName) > 0 {
		errDeletingConfigMap := r.configMapInterfaces[msg.namespace].Delete(
			context.TODO(), msg.configMapName,
			metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if errDeletingConfigMap != nil {
			r.syslog.WithError(err).Errorf("failed to delete configMap %s", msg.configMapName)
			err = errDeletingConfigMap
		} else {
			r.syslog.Infof("deleted configMap %s", msg.configMapName)
		}
	}

	// It is possible that the creator of the message is no longer around.
	// However this should have no impact on correctness.
	if err != nil {
		go msg.errorHandler(resourceDeletionFailed{err})
	}
}
