package kubernetesrm

import (
	"context"
	"fmt"

	batchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"

	"github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/ptrs"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type requestProcessingWorker struct {
	jobInterface        map[string]batchV1.JobInterface
	podInterface        map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface
	failures            chan<- resourcesRequestFailure
	syslog              *logrus.Entry
}

type readyCallbackFunc func(createRef requestID)

func startRequestProcessingWorker(
	jobInterface map[string]batchV1.JobInterface,
	podInterface map[string]typedV1.PodInterface,
	configMapInterfaces map[string]typedV1.ConfigMapInterface,
	id string,
	in <-chan interface{},
	ready readyCallbackFunc,
	failures chan<- resourcesRequestFailure,
) *requestProcessingWorker {
	syslog := logrus.WithField("component", "kubernetesrm-worker").WithField("id", id)
	r := &requestProcessingWorker{
		jobInterface:        jobInterface,
		podInterface:        podInterface,
		configMapInterfaces: configMapInterfaces,
		failures:            failures,
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
			go ready(keyForCreate(msg))
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
	r.syslog.Debugf("creating configMap %v", msg.configMapSpec.Name)
	configMap, err := r.configMapInterfaces[msg.jobSpec.Namespace].Create(
		context.TODO(), msg.configMapSpec, metaV1.CreateOptions{})
	if err != nil {
		r.syslog.WithError(err).Errorf("error creating configMap %s", msg.configMapSpec.Name)
		r.failures <- resourceCreationFailed{jobName: msg.jobSpec.Name, err: err}
		return
	}
	r.syslog.Infof("created configMap %s", configMap.Name)

	r.syslog.Debugf("creating job %s", msg.jobSpec.Name)
	job, err := r.jobInterface[msg.jobSpec.Namespace].Create(
		context.TODO(), msg.jobSpec, metaV1.CreateOptions{},
	)
	if err != nil {
		r.syslog.WithError(err).Errorf("error creating job %s", msg.jobSpec.Name)
		r.failures <- resourceCreationFailed{jobName: msg.jobSpec.Name, err: err}
		return
	}
	r.syslog.Infof("created job %s", job.Name)
}

func (r *requestProcessingWorker) receiveDeleteKubernetesResources(
	msg deleteKubernetesResources,
) {
	var gracePeriod int64 = deletionGracePeriod
	var err error

	// If resource creation failed, we will still try to delete those resources which
	// will also result in a failure.
	if len(msg.jobName) > 0 {
		err = r.jobInterface[msg.namespace].Delete(context.TODO(), msg.jobName, metaV1.DeleteOptions{
			GracePeriodSeconds: &gracePeriod,
			PropagationPolicy:  ptrs.Ptr(metaV1.DeletePropagationBackground),
		})
		if err != nil {
			r.syslog.WithError(err).Errorf("failed to delete pod %s", msg.jobName)
		} else {
			r.syslog.Infof("deleted job %s", msg.jobName)
		}
	}

	if len(msg.podName) > 0 {
		err = r.podInterface[msg.namespace].Delete(
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
		r.failures <- resourceDeletionFailed{jobName: msg.jobName, err: err}
	}
}
