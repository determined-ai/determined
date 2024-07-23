package kubernetesrm

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	batchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	alphaGateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1alpha2"

	"github.com/determined-ai/determined/master/pkg/ptrs"
)

type requestProcessingWorker struct {
	jobInterface        map[string]batchV1.JobInterface
	podInterface        map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface
	serviceInterfaces   map[string]typedV1.ServiceInterface
	tcpRouteInterfaces  map[string]alphaGateway.TCPRouteInterface
	gatewayService      *gatewayService
	failures            chan<- resourcesRequestFailure
	syslog              *logrus.Entry
}

type readyCallbackFunc func(createRef requestID)

func startRequestProcessingWorker(
	jobInterface map[string]batchV1.JobInterface,
	podInterface map[string]typedV1.PodInterface,
	configMapInterfaces map[string]typedV1.ConfigMapInterface,
	serviceInterfaces map[string]typedV1.ServiceInterface,
	gatewayService *gatewayService,
	tcpRouteInterfaces map[string]alphaGateway.TCPRouteInterface,
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
		serviceInterfaces:   serviceInterfaces,
		gatewayService:      gatewayService,
		tcpRouteInterfaces:  tcpRouteInterfaces,

		failures: failures,
		syslog:   syslog,
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

	var ports []int
	var proxyResources []gatewayProxyResource
	// TODO(RM-272) do we leak resources if the request queue fails?
	// Do we / should we delete created resources?
	if msg.gw != nil {
		if msg.gw.requestedPorts > 0 {
			if ports, err = r.gatewayService.generateAndAddListeners(
				msg.gw.allocationID, msg.gw.requestedPorts,
			); err != nil {
				r.syslog.WithError(err).Errorf("error patching gateway for job %s", msg.jobSpec.Name)
				r.failures <- resourceCreationFailed{jobName: msg.jobSpec.Name, err: err}
				return
			}
			r.syslog.Info("created gateway proxy listeners", msg.gw.requestedPorts, ports)
		}
		proxyResources = msg.gw.resourceDescriptor(ports)
	}

	for _, proxyResource := range proxyResources {
		r.syslog.Debugf("launching service with spec %v", *proxyResource.serviceSpec)
		if _, err := r.serviceInterfaces[msg.jobSpec.Namespace].Create(
			context.TODO(), proxyResource.serviceSpec, metaV1.CreateOptions{},
		); err != nil {
			r.syslog.WithError(err).Errorf("error creating service for pod %s", msg.jobSpec.Name)
			r.failures <- resourceCreationFailed{jobName: msg.jobSpec.Name, err: err}
			return
		}
		r.syslog.Debugf("launching tcproute with spec %v", *proxyResource.tcpRouteSpec)
		if _, err := r.tcpRouteInterfaces[msg.jobSpec.Namespace].Create(
			context.TODO(), proxyResource.tcpRouteSpec, metaV1.CreateOptions{},
		); err != nil {
			r.syslog.WithError(err).Errorf("error creating tcproute for pod %s", msg.jobSpec.Name)
			r.failures <- resourceCreationFailed{jobName: msg.jobSpec.Name, err: err}
			return
		}
	}
	if msg.gw != nil && msg.gw.reportResources != nil {
		msg.gw.reportResources(proxyResources)
		r.syslog.Info("created gateway proxy resources")
	}
}

func (r *requestProcessingWorker) receiveDeleteKubernetesResources(
	msg deleteKubernetesResources,
) {
	var gracePeriod int64 = deletionGracePeriod
	var err error

	// If resource creation failed, we will still try to delete those resources which
	// will also result in a failure.
	if len(msg.jobName) > 0 {
		_, ok := r.jobInterface[msg.namespace]
		if ok {
			err = r.jobInterface[msg.namespace].Delete(context.TODO(), msg.jobName,
				metaV1.DeleteOptions{
					GracePeriodSeconds: &gracePeriod,
					PropagationPolicy:  ptrs.Ptr(metaV1.DeletePropagationBackground),
				})
			switch {
			case k8serrors.IsNotFound(err):
				r.syslog.Infof("job %s is already deleted", msg.jobName)
			case err != nil:
				r.syslog.WithError(err).Errorf("failed to delete job %s", msg.jobName)
			default:
				r.syslog.Infof("deleted job %s", msg.jobName)
			}
		}
	}

	if len(msg.podName) > 0 {
		_, ok := r.podInterface[msg.namespace]
		if ok {
			err = r.podInterface[msg.namespace].Delete(
				context.TODO(), msg.podName, metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
			switch {
			case k8serrors.IsNotFound(err):
				r.syslog.Infof("pod %s is already deleted", msg.jobName)
			case err != nil:
				r.syslog.WithError(err).Errorf("failed to delete pod %s", msg.jobName)
			default:
				r.syslog.Infof("deleted pod %s", msg.podName)
			}
		}
	}

	if len(msg.configMapName) > 0 {
		_, ok := r.configMapInterfaces[msg.namespace]
		if ok {
			err = r.configMapInterfaces[msg.namespace].Delete(
				context.TODO(), msg.configMapName,
				metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
			switch {
			case k8serrors.IsNotFound(err):
				r.syslog.Infof("configMap %s is already deleted", msg.jobName)
			case err != nil:
				r.syslog.WithError(err).Errorf("failed to delete configMap %s", msg.jobName)
			default:
				r.syslog.Infof("deleted configMap %s", msg.configMapName)
			}
		}
	}

	for _, serviceName := range msg.serviceNames {
		errDeletingService := r.serviceInterfaces[msg.namespace].Delete(
			context.TODO(), serviceName,
			metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if errDeletingService != nil {
			r.syslog.WithError(errDeletingService).
				Errorf("failed to delete service %s", serviceName)
			err = errDeletingService
		} else {
			r.syslog.Infof("deleted service %s", serviceName)
		}
	}

	for _, tcpRouteName := range msg.tcpRouteNames {
		errDeletingService := r.tcpRouteInterfaces[msg.namespace].Delete(
			context.TODO(), tcpRouteName,
			metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if errDeletingService != nil {
			r.syslog.WithError(errDeletingService).
				Errorf("failed to delete tcpRoute %s", tcpRouteName)
			err = errDeletingService
		} else {
			r.syslog.Infof("deleted tcpRoute %s", tcpRouteName)
		}
	}

	if len(msg.gatewayPortsToFree) > 0 {
		err = r.gatewayService.freePorts(msg.gatewayPortsToFree)
		if err != nil {
			r.syslog.WithError(err).Errorf("failed to free gateway ports %v", msg.gatewayPortsToFree)
		} else {
			r.syslog.Infof("freed gateway ports %v", msg.gatewayPortsToFree)
		}
	}

	// It is possible that the creator of the message is no longer around.
	// However this should have no impact on correctness.
	if err != nil {
		r.failures <- resourceDeletionFailed{jobName: msg.jobName, err: err}
	}
}
