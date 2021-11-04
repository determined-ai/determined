package kubernetes

import (
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"

	k8sV1 "k8s.io/api/core/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	numKubernetesWorkers = 5
	deletionGracePeriod  = 15
)

// message types that are sent to the requestQueue.
type (
	createKubernetesResources struct {
		handler       *actor.Ref
		podSpec       *k8sV1.Pod
		configMapSpec *k8sV1.ConfigMap
	}

	deleteKubernetesResources struct {
		handler       *actor.Ref
		podName       string
		configMapName string
	}
)

// message types that are sent by requestQueue and requestProcessingWorkers as responses
// to creation or deletion requests.
type (
	resourceCreationFailed struct {
		err error
	}

	resourceDeletionFailed struct {
		err error
	}

	resourceCreationCancelled struct{}
)

// message types sent from requestProcessingWorkers to requestQueue.
type (
	workerAvailable struct {
		resourceHandler *actor.Ref
	}
)

// queuedResourceRequest is used to represent requests that are being buffered by requestQueue.
type queuedResourceRequest struct {
	createResources *createKubernetesResources
	deleteResources *deleteKubernetesResources
}

// The requestQueue is responsible for fulfilling all requests for creating and deleting
// kubernetes resources that require interaction with the kubernetes API. It accomplishes
// this by forwarding requests to requestProcessingWorker actors which prcess the request.
// There are two reasons a queue system is required as opposed to allowing the pod actors
// to create and delete Kubernetes resources asynchronously themselves:
//
//    1) Each pod creation first requires the creation of a configMap, however creating the two
//       is not an atomic operation. If there is a large number of concurrent creation requests
//       (e.g., a large HP search experiment) the kubernetes API server ends up processing the
//       creation of all the configMaps before starting to create pods, which adds significant
//       latency to the creation of pods.
//
//    2) If all creation and deletion requests are submitted asynchronously, it is possible the
//       Kubernetes API server will temporarily become saturated, and be slower to respond to other
//       requests.
//
//  When requests come in they are buffered by the requestQueue until a worker becomes available
//  at which point the longest queue request is forwarded to the available. Requests are buffered
//  rather than forward right away because buffering makes it possible to cancel creation requests
//  after they are created, but before they are executed. Since the actor system processes messages
//  in a FIFO order, if all request were forwarded right away any cancellation request would only
//  be processed after the creation request case already been processed, requiring an unnecessary
//  resource creation and deletion. An example of this is when a large HP search is created and
//  then killed moments later. By having requests be buffered, if a deletion request arrives
//  prior to the creation request being executed, the requestQueue detects this and skips the
//  unnecessary creation / deletion.
//
//  The message protocol consists of `createKubernetesResources` and `deleteKubernetesResources`
//  messages being sent to the requestQueue. If it forwards the request to a worker, the worker
//  will send the original task handler a `resourceCreationFailed` or a `resourceDeletionFailed`
//  if an error was encountered while creating / deleting the resources. If a deletion request
//  arrives before the creation request had been sent to the worker, the `requestQueue` will
//  notify the task handler of this by sending a `resourceCreationCancelled` message.
//  requestProcessingWorkers notify the requestQueue that they are available to receive work
//  by sending a `workerAvailable` message.
type requestQueue struct {
	podInterface       typedV1.PodInterface
	configMapInterface typedV1.ConfigMapInterface

	queue                    []*queuedResourceRequest
	pendingResourceCreations map[*actor.Ref]*queuedResourceRequest
	availableWorkers         []*actor.Ref

	creationInProgress       map[*actor.Ref]bool
	blockedResourceDeletions map[*actor.Ref]*queuedResourceRequest
}

func newRequestQueue(
	podInterface typedV1.PodInterface,
	configMapInterface typedV1.ConfigMapInterface,
) *requestQueue {
	return &requestQueue{
		podInterface:       podInterface,
		configMapInterface: configMapInterface,

		queue:                    make([]*queuedResourceRequest, 0),
		pendingResourceCreations: make(map[*actor.Ref]*queuedResourceRequest),
		availableWorkers:         make([]*actor.Ref, 0, numKubernetesWorkers),

		creationInProgress:       make(map[*actor.Ref]bool),
		blockedResourceDeletions: make(map[*actor.Ref]*queuedResourceRequest),
	}
}

func (r *requestQueue) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		for i := 0; i < numKubernetesWorkers; i++ {
			newWorker, ok := ctx.ActorOf(
				fmt.Sprintf("kubernetes-worker-%d", i),
				&requestProcessingWorker{
					podInterface:       r.podInterface,
					configMapInterface: r.configMapInterface,
				},
			)
			if !ok {
				return errors.Errorf("%s actor already exists", newWorker.Address())
			}
		}
	case actor.PostStop:
		// This should not happen since the request queue actor would not stop during
		// the master is running.

	case createKubernetesResources:
		r.receiveCreateKubernetesResources(ctx, msg)

	case deleteKubernetesResources:
		r.receiveDeleteKubernetesResources(ctx, msg)

	case workerAvailable:
		r.receiveWorkerAvailable(ctx, msg)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}

	return nil
}

func (r *requestQueue) receiveCreateKubernetesResources(
	ctx *actor.Context,
	msg createKubernetesResources,
) {
	if _, requestAlreadyExists := r.pendingResourceCreations[msg.handler]; requestAlreadyExists {
		ctx.Log().Errorf(
			"actor %s issued multiple request requests to create kubernetes resources",
			msg.handler.Address())
		return
	}

	if len(r.availableWorkers) > 0 {
		r.creationInProgress[msg.handler] = true
		ctx.Tell(r.availableWorkers[0], msg)
		r.availableWorkers = r.availableWorkers[1:]
		return
	}

	queuedRequest := &queuedResourceRequest{createResources: &msg}
	r.queue = append(r.queue, queuedRequest)
	r.pendingResourceCreations[msg.handler] = queuedRequest
}

func (r *requestQueue) receiveDeleteKubernetesResources(
	ctx *actor.Context,
	msg deleteKubernetesResources,
) {
	// If the request has not been processed yet, cancel it and inform the handler.
	if _, creationPending := r.pendingResourceCreations[msg.handler]; creationPending {
		r.pendingResourceCreations[msg.handler].createResources = nil
		delete(r.pendingResourceCreations, msg.handler)
		ctx.Tell(msg.handler, resourceCreationCancelled{})
		return
	}

	// We do not want to trigger resource deletion concurrently with resource creation.
	// If the creation request is currently being processed, we delay processing the
	// deletion request.
	if _, creationInProgress := r.creationInProgress[msg.handler]; creationInProgress {
		r.blockedResourceDeletions[msg.handler] = &queuedResourceRequest{deleteResources: &msg}
		return
	}

	if len(r.availableWorkers) > 0 {
		ctx.Tell(r.availableWorkers[0], msg)
		r.availableWorkers = r.availableWorkers[1:]
		return
	}

	r.queue = append(r.queue, &queuedResourceRequest{deleteResources: &msg})
}

func (r *requestQueue) receiveWorkerAvailable(ctx *actor.Context, msg workerAvailable) {
	if msg.resourceHandler != nil {
		delete(r.creationInProgress, msg.resourceHandler)

		// Check if any deletions were blocked by this creation.
		queuedMsg, resourceDeletionWasBlocked := r.blockedResourceDeletions[msg.resourceHandler]
		if resourceDeletionWasBlocked {
			r.queue = append(r.queue, queuedMsg)
			delete(r.blockedResourceDeletions, msg.resourceHandler)
		}
	}

	for len(r.queue) > 0 {
		nextRequest := r.queue[0]
		r.queue = r.queue[1:]

		// If both creation and deletion are nil it means that the creation
		// request was canceled.
		if nextRequest.createResources != nil {
			delete(r.pendingResourceCreations, nextRequest.createResources.handler)
			r.creationInProgress[nextRequest.createResources.handler] = true
			ctx.Tell(ctx.Sender(), *nextRequest.createResources)
			return
		} else if nextRequest.deleteResources != nil {
			ctx.Tell(ctx.Sender(), *nextRequest.deleteResources)
			return
		}
	}

	r.availableWorkers = append(r.availableWorkers, ctx.Sender())
}
