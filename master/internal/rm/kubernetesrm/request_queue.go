package kubernetesrm

import (
	"strconv"
	"strings"
	"sync"

	batchV1 "k8s.io/api/batch/v1"
	typedBatchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"

	"github.com/sirupsen/logrus"
	k8sV1 "k8s.io/api/core/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	alphaGateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1alpha2"

	"github.com/determined-ai/determined/master/pkg/set"
)

const (
	numKubernetesWorkers = 5
	deletionGracePeriod  = 15
)

// message types that are sent to the requestProcessingWorkers channel.
type (
	createKubernetesResources struct {
		jobSpec       *batchV1.Job
		configMapSpec *k8sV1.ConfigMap
		gw            *gatewayResourceComm
	}

	deleteKubernetesResources struct {
		namespace          string
		jobName            string
		podName            string
		configMapName      string
		serviceNames       []string
		tcpRouteNames      []string
		gatewayPortsToFree []int
	}
)

// error types that are sent by requestQueue and requestProcessingWorkers as responses
// to creation or deletion requests.
type (
	resourceCreationFailed struct {
		jobName string
		err     error
	}
	resourceDeletionFailed struct {
		jobName string
		err     error
	}
	resourceCreationCancelled struct {
		jobName string
	}
)

type resourcesRequestFailure interface {
	getJobName() string
	resourcesRequestFailure()
}

func (e resourceCreationFailed) getJobName() string    { return e.jobName }
func (e resourceDeletionFailed) getJobName() string    { return e.jobName }
func (e resourceCreationCancelled) getJobName() string { return e.jobName }

func (resourceCreationFailed) resourcesRequestFailure()    {}
func (resourceDeletionFailed) resourcesRequestFailure()    {}
func (resourceCreationCancelled) resourcesRequestFailure() {}

// queuedResourceRequest is used to represent requests that are being buffered by requestQueue.
type queuedResourceRequest struct {
	createResources *createKubernetesResources
	deleteResources *deleteKubernetesResources
}

// The requestQueue is responsible for fulfilling all requests for creating and deleting
// kubernetes resources that require interaction with the kubernetes API. It accomplishes
// this by forwarding requests to requestProcessingWorker goroutines which process the request.
// There are two reasons a queue system is required as opposed to allowing the pod routines
// to create and delete Kubernetes resources asynchronously themselves:
//
//  1. Each pod creation first requires the creation of a configMap, however creating the two
//     is not an atomic operation. If there is a large number of concurrent creation requests
//     (e.g., a large HP search experiment) the kubernetes API server ends up processing the
//     creation of all the configMaps before starting to create pods, which adds significant
//     latency to the creation of pods.
//
//  2. If all creation and deletion requests are submitted asynchronously, it is possible the
//     Kubernetes API server will temporarily become saturated, and be slower to respond to other
//     requests.
//
//     When requests come in they are buffered by the requestQueue until a worker becomes available
//     at which point the oldest queued request is forwarded to the worker. Requests are buffered
//     rather than forward right away because buffering makes it possible to cancel creation
//     requests after they are created, but before they are executed. Since the queue locking
//     processes messages in a FIFO order, if all request were forwarded right away any cancellation
//     request would only be processed after the creation request case already been processed,
//     requiring an unnecessary resource creation and deletion. An example of this is when a
//     large HP search is created and then killed moments later. By having requests be buffered,
//     if a deletion request arrives prior to the creation request being executed,
//     the requestQueue detects this and skips the unnecessary creation / deletion.
//
//     The message protocol consists of `createKubernetesResources` and `deleteKubernetesResources`
//     messages being sent to the requestQueue. If it forwards the request to a worker, the worker
//     will send the original task handler a `resourceCreationFailed` or a `resourceDeletionFailed`
//     if an error was encountered while creating / deleting the resources. If a deletion request
//     arrives before the creation request had been sent to the worker, the `requestQueue` will
//     notify the task handler of this by sending a `resourceCreationCancelled` message.
//     requestProcessingWorkers notify the requestQueue that they are available to receive work
//     by sending a `workerAvailable` message.
type requestQueue struct {
	jobInterfaces       map[string]typedBatchV1.JobInterface
	podInterfaces       map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface
	serviceInterfaces   map[string]typedV1.ServiceInterface
	tcpRouteInterfaces  map[string]alphaGateway.TCPRouteInterface
	gatewayService      *gatewayService

	failures chan<- resourcesRequestFailure

	mu         sync.Mutex
	workerChan chan interface{}

	queue []*queuedResourceRequest

	creationInProgress       set.Set[requestID]
	pendingResourceCreations map[requestID]*queuedResourceRequest
	blockedResourceDeletions map[requestID]*queuedResourceRequest

	syslog *logrus.Entry
}

type requestID string

func startRequestQueue(
	jobInterfaces map[string]typedBatchV1.JobInterface,
	podInterfaces map[string]typedV1.PodInterface,
	configMapInterfaces map[string]typedV1.ConfigMapInterface,
	serviceInterfaces map[string]typedV1.ServiceInterface,
	gatewayService *gatewayService,
	tcpRouteInterfaces map[string]alphaGateway.TCPRouteInterface,
	failures chan<- resourcesRequestFailure,
) (*requestQueue, []*requestProcessingWorker) {
	r := &requestQueue{
		jobInterfaces:       jobInterfaces,
		podInterfaces:       podInterfaces,
		configMapInterfaces: configMapInterfaces,
		serviceInterfaces:   serviceInterfaces,
		gatewayService:      gatewayService,
		tcpRouteInterfaces:  tcpRouteInterfaces,

		failures: failures,

		workerChan: make(chan interface{}),

		queue: make([]*queuedResourceRequest, 0),

		creationInProgress:       make(set.Set[requestID]),
		pendingResourceCreations: make(map[requestID]*queuedResourceRequest),
		blockedResourceDeletions: make(map[requestID]*queuedResourceRequest),

		syslog: logrus.WithField("component", "kubernetesrm-queue"),
	}
	workers := r.startWorkers()
	return r, workers
}

func (r *requestQueue) startWorkers() []*requestProcessingWorker {
	allWorkers := []*requestProcessingWorker{}
	for i := 0; i < numKubernetesWorkers; i++ {
		worker := startRequestProcessingWorker(
			r.jobInterfaces,
			r.podInterfaces,
			r.configMapInterfaces,
			r.serviceInterfaces,
			r.gatewayService,
			r.tcpRouteInterfaces,
			strconv.Itoa(i),
			r.workerChan,
			r.workerReady,
			r.failures,
		)
		allWorkers = append(allWorkers, worker)
	}
	return allWorkers
}

func keyForCreate(msg createKubernetesResources) requestID {
	if msg.jobSpec != nil {
		return requestID(msg.jobSpec.Namespace + "/" + msg.jobSpec.Name)
	}
	if msg.configMapSpec != nil {
		return requestID(msg.configMapSpec.Namespace + "/" + msg.configMapSpec.Name)
	}
	panic("invalid createKubernetesResources message")
}

func keyForDelete(msg deleteKubernetesResources) requestID {
	if msg.jobName != "" {
		return requestID(msg.namespace + "/" + msg.jobName)
	}
	if msg.podName != "" {
		return requestID(msg.namespace + "/" + msg.podName)
	}
	if msg.configMapName != "" {
		return requestID(msg.namespace + "/" + msg.configMapName)
	}
	if len(msg.serviceNames) > 0 {
		return requestID(msg.namespace + "/" + strings.Join(msg.serviceNames, ","))
	}
	if len(msg.tcpRouteNames) > 0 {
		return requestID(msg.namespace + "/" + strings.Join(msg.tcpRouteNames, ","))
	}
	if len(msg.gatewayPortsToFree) > 0 {
		var stringPorts []string
		for _, p := range msg.gatewayPortsToFree {
			stringPorts = append(stringPorts, strconv.Itoa(p))
		}

		return requestID(msg.namespace + "/" + strings.Join(stringPorts, ","))
	}

	panic("invalid deleteKubernetesResources message")
}

func (r *requestQueue) createKubernetesResources(
	jobSpec *batchV1.Job,
	configMapSpec *k8sV1.ConfigMap,
	gwResources *gatewayResourceComm,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	msg := createKubernetesResources{jobSpec, configMapSpec, gwResources}
	ref := keyForCreate(msg)

	if _, requestAlreadyExists := r.pendingResourceCreations[ref]; requestAlreadyExists {
		r.syslog.Errorf("multiple create resource requests issued for %s", ref)
		return
	}

	select {
	case r.workerChan <- msg:
		r.creationInProgress.Insert(ref)
	default:
		queuedRequest := &queuedResourceRequest{createResources: &msg}
		r.queue = append(r.queue, queuedRequest)
		r.pendingResourceCreations[ref] = queuedRequest
	}
}

func (r *requestQueue) deleteKubernetesResources(msg deleteKubernetesResources) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ref := keyForDelete(msg)

	// If the request has not been processed yet, cancel it and inform the handler.
	if _, creationPending := r.pendingResourceCreations[ref]; creationPending {
		r.pendingResourceCreations[ref].createResources = nil
		delete(r.pendingResourceCreations, ref)
		r.failures <- resourceCreationCancelled{
			jobName: msg.jobName,
		}
		r.syslog.Warnf("delete issued with pending create request for %s", ref)
		return
	}

	// We do not want to trigger resource deletion concurrently with resource creation.
	// If the creation request is currently being processed, we delay processing the
	// deletion request.
	if r.creationInProgress.Contains(ref) {
		r.blockedResourceDeletions[ref] = &queuedResourceRequest{deleteResources: &msg}
		return
	}

	select {
	case r.workerChan <- msg:
	default:
		r.queue = append(r.queue, &queuedResourceRequest{deleteResources: &msg})
	}
}

func (r *requestQueue) workerReady(createRef requestID) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if createRef != "" {
		r.creationInProgress.Remove(createRef)

		// Check if any deletions were blocked by this creation.
		if queuedMsg, ok := r.blockedResourceDeletions[createRef]; ok {
			r.queue = append(r.queue, queuedMsg)
			delete(r.blockedResourceDeletions, createRef)
		}
	}

	for len(r.queue) > 0 {
		nextRequest := r.queue[0]
		r.queue = r.queue[1:]

		// If both creation and deletion are nil it means that the creation
		// request was canceled.
		if nextRequest.createResources != nil {
			next := keyForCreate(*nextRequest.createResources)
			delete(r.pendingResourceCreations, next)
			r.creationInProgress.Insert(next)
			r.workerChan <- *nextRequest.createResources
			return
		} else if nextRequest.deleteResources != nil {
			r.workerChan <- *nextRequest.deleteResources
			return
		}
	}
}
