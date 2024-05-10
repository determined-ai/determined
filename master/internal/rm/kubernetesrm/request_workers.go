package kubernetesrm

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"
	alphaGateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1alpha2"
)

type requestProcessingWorker struct {
	podInterfaces       map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface
	serviceInterfaces   map[string]typedV1.ServiceInterface
	tcpRouteInterfaces  map[string]alphaGateway.TCPRouteInterface
	gatewayService      *gatewayService
	failures            chan<- resourcesRequestFailure
	syslog              *logrus.Entry
}

type readyCallbackFunc func(createRef requestID)

func startRequestProcessingWorker(
	podInterfaces map[string]typedV1.PodInterface,
	configMapInterfaces map[string]typedV1.ConfigMapInterface,
	serviceInterfaces map[string]typedV1.ServiceInterface,
	gatewayService *gatewayService,
	tcpRouteInterfaces map[string]alphaGateway.TCPRouteInterface,
	id string,
	in <-chan interface{},
	ready readyCallbackFunc,
	failures chan<- resourcesRequestFailure,
) *requestProcessingWorker {
	syslog := logrus.New().WithField("component", "kubernetesrm-worker").WithField("id", id)
	r := &requestProcessingWorker{
		podInterfaces:       podInterfaces,
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
	r.syslog.Debugf("creating configMap with spec %v", msg.configMapSpec)
	configMap, err := r.configMapInterfaces[msg.podSpec.Namespace].Create(
		context.TODO(), msg.configMapSpec, metaV1.CreateOptions{})
	if err != nil {
		r.syslog.WithError(err).Errorf("error creating configMap %s", msg.configMapSpec.Name)
		r.failures <- resourceCreationFailed{podName: msg.podSpec.Name, err: err}
		return
	}
	r.syslog.Infof("created configMap %s", configMap.Name)

	var gatewayListeners []gatewayTyped.Listener
	for _, proxyResource := range msg.gatewayProxyResources {
		r.syslog.Debugf("launching service with spec %v", *proxyResource.serviceSpec)
		if _, err := r.serviceInterfaces[msg.podSpec.Namespace].Create(
			context.TODO(), proxyResource.serviceSpec, metaV1.CreateOptions{},
		); err != nil {
			r.syslog.WithError(err).Errorf("error creating service for pod %s", msg.podSpec.Name)
			r.failures <- resourceCreationFailed{podName: msg.podSpec.Name, err: err}
			return
		}

		r.syslog.Debugf("launching tcproute with spec %v", *proxyResource.tcpRouteSpec)
		if _, err := r.tcpRouteInterfaces[msg.podSpec.Namespace].Create(
			context.TODO(), proxyResource.tcpRouteSpec, metaV1.CreateOptions{},
		); err != nil {
			r.syslog.WithError(err).Errorf("error creating tcproute for pod %s", msg.podSpec.Name)
			r.failures <- resourceCreationFailed{podName: msg.podSpec.Name, err: err}
			return
		}

		gatewayListeners = append(gatewayListeners, proxyResource.gatewayListener)
	}
	if len(gatewayListeners) > 0 {
		// TODO(RM-272) do we leak resources if the request queue fails?
		// Do we / should we delete created resources?
		if err := r.gatewayService.addListeners(gatewayListeners); err != nil {
			r.syslog.WithError(err).Errorf("error patching gateway for pod %s", msg.podSpec.Name)
			r.failures <- resourceCreationFailed{podName: msg.podSpec.Name, err: err}
			return
		}
		r.syslog.Info("created gateway proxy resources")
	}

	r.syslog.Debugf("launching pod with spec %v", msg.podSpec)
	pod, err := r.podInterfaces[msg.podSpec.Namespace].Create(
		context.TODO(), msg.podSpec, metaV1.CreateOptions{},
	)
	if err != nil {
		r.syslog.WithError(err).Errorf("error creating pod %s", msg.podSpec.Name)
		r.failures <- resourceCreationFailed{podName: msg.podSpec.Name, err: err}
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
	if podName := msg.podName; podName != nil {
		err = r.podInterfaces[msg.namespace].Delete(
			context.TODO(), *podName, metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if err != nil {
			r.syslog.WithError(err).Errorf("failed to delete pod %s", *podName)
		} else {
			r.syslog.Infof("deleted pod %s", *podName)
		}
	}

	if configMapName := msg.configMapName; configMapName != nil {
		errDeletingConfigMap := r.configMapInterfaces[msg.namespace].Delete(
			context.TODO(), *configMapName,
			metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
		if errDeletingConfigMap != nil {
			r.syslog.WithError(errDeletingConfigMap).
				Errorf("failed to delete configMap %s", *configMapName)
			err = errDeletingConfigMap
		} else {
			r.syslog.Infof("deleted configMap %s", *configMapName)
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
		podName := "" // TODO(RM-272): this code is strange to me since podName can be empty.
		if msg.podName != nil {
			podName = *msg.podName
		}

		r.failures <- resourceDeletionFailed{podName: podName, err: err}
	}
}
