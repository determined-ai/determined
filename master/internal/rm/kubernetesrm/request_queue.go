package kubernetesrm

import (
	"reflect"
	"strconv"
	"sync"

	"github.com/sirupsen/logrus"
	k8sV1 "k8s.io/api/core/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/pkg/set"
)

const (
	numKubernetesWorkers = 5
	deletionGracePeriod  = 15
)

// callback types used by requestQueue.
type (
	errorCallbackFunc func(error)
	callbackPtr       reflect.Value
)

// message types that are sent to the requestProcessingWorkers channel.
type (
	createKubernetesResources struct {
		errorHandler  errorCallbackFunc
		podSpec       *k8sV1.Pod
		configMapSpec *k8sV1.ConfigMap
	}

	deleteKubernetesResources struct {
		errorHandler  errorCallbackFunc
		namespace     string
		podName       string
		configMapName string
	}
)

// error types that are sent by requestQueue and requestProcessingWorkers as responses
// to creation or deletion requests.
type (
	resourceCreationFailed    struct{ error }
	resourceDeletionFailed    struct{ error }
	resourceCreationCancelled struct{ error }
)

// queuedResourceRequest is used to represent requests that are being buffered by requestQueue.
type queuedResourceRequest struct {
	createResources *createKubernetesResources
	deleteResources *deleteKubernetesResources
}

// The requestQueue is responsible for fulfilling all requests for creating and deleting
// kubernetes resources that require interaction with the kubernetes API. It accomplishes
// this by forwarding requests to requestProcessingWorker go routines which process the request.
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
//     at which point the longest queue request is forwarded to the available. Requests are buffered
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
	podInterfaces       map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface

	mu         sync.Mutex
	workerChan chan interface{}

	queue []*queuedResourceRequest

	creationInProgress       set.Set[callbackPtr]
	pendingResourceCreations map[callbackPtr]*queuedResourceRequest
	blockedResourceDeletions map[callbackPtr]*queuedResourceRequest

	availableWorkers int

	syslog *logrus.Entry
}

func getCallbackPtr(handler errorCallbackFunc) callbackPtr {
	return (callbackPtr)(reflect.ValueOf(handler))
}

func startRequestQueue(
	podInterfaces map[string]typedV1.PodInterface,
	configMapInterfaces map[string]typedV1.ConfigMapInterface,
) *requestQueue {
	r := &requestQueue{
		podInterfaces:       podInterfaces,
		configMapInterfaces: configMapInterfaces,

		workerChan: make(chan interface{}),

		queue: make([]*queuedResourceRequest, 0),

		creationInProgress:       make(set.Set[callbackPtr]),
		pendingResourceCreations: make(map[callbackPtr]*queuedResourceRequest),
		blockedResourceDeletions: make(map[callbackPtr]*queuedResourceRequest),

		syslog: logrus.New().WithField("component", "kubernetesrm-queue"),
	}
	r.startWorkers()
	return r
}

func (r *requestQueue) startWorkers() {
	for i := 0; i < numKubernetesWorkers; i++ {
		startRequestProcessingWorker(
			r.podInterfaces,
			r.configMapInterfaces,
			strconv.Itoa(i),
			r.workerChan,
			r.workerAvailableCallback,
		)
	}
}

func (r *requestQueue) createKubernetesResources(
	errorHandler errorCallbackFunc,
	podSpec *k8sV1.Pod,
	configMapSpec *k8sV1.ConfigMap,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ref := getCallbackPtr(errorHandler)

	if _, requestAlreadyExists := r.pendingResourceCreations[ref]; requestAlreadyExists {
		r.syslog.Errorf("handler %v issued multiple create resource requests", errorHandler)
		return
	}

	msg := createKubernetesResources{errorHandler, podSpec, configMapSpec}
	select {
	case r.workerChan <- msg:
		r.creationInProgress.Insert(ref)
		r.availableWorkers--
	default:
		queuedRequest := &queuedResourceRequest{createResources: &msg}
		r.queue = append(r.queue, queuedRequest)
		r.pendingResourceCreations[ref] = queuedRequest
	}
}

func (r *requestQueue) deleteKubernetesResources(
	errorHandler errorCallbackFunc,
	namespace string,
	podName string,
	configMapName string,
) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ref := getCallbackPtr(errorHandler)

	// If the request has not been processed yet, cancel it and inform the handler.
	if _, creationPending := r.pendingResourceCreations[ref]; creationPending {
		r.pendingResourceCreations[ref].createResources = nil
		delete(r.pendingResourceCreations, ref)
		go errorHandler(resourceCreationCancelled{})
		r.syslog.Warnf("handler %v issued delete with pending create request", errorHandler)
		return
	}

	msg := deleteKubernetesResources{errorHandler, namespace, podName, configMapName}
	// We do not want to trigger resource deletion concurrently with resource creation.
	// If the creation request is currently being processed, we delay processing the
	// deletion request.
	if r.creationInProgress.Contains(ref) {
		r.blockedResourceDeletions[ref] = &queuedResourceRequest{deleteResources: &msg}
		return
	}

	select {
	case r.workerChan <- msg:
		r.availableWorkers--
	default:
		r.queue = append(r.queue, &queuedResourceRequest{deleteResources: &msg})
	}
}

func (r *requestQueue) workerAvailableCallback(errorHandler errorCallbackFunc) {
	r.mu.Lock()
	defer r.mu.Unlock()
	ref := getCallbackPtr(errorHandler)

	if errorHandler != nil {
		r.creationInProgress.Remove(ref)

		// Check if any deletions were blocked by this creation.
		queuedMsg, resourceDeletionWasBlocked := r.blockedResourceDeletions[ref]
		if resourceDeletionWasBlocked {
			r.queue = append(r.queue, queuedMsg)
			delete(r.blockedResourceDeletions, ref)
		}
	}

	for len(r.queue) > 0 {
		nextRequest := r.queue[0]
		r.queue = r.queue[1:]

		// If both creation and deletion are nil it means that the creation
		// request was canceled.
		if nextRequest.createResources != nil {
			next := getCallbackPtr(nextRequest.createResources.errorHandler)
			delete(r.pendingResourceCreations, next)
			r.creationInProgress.Insert(next)
			r.workerChan <- *nextRequest.createResources
			return
		} else if nextRequest.deleteResources != nil {
			r.workerChan <- *nextRequest.deleteResources
			return
		}
	}
	r.availableWorkers++
}
