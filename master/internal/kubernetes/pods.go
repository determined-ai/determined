// Package kubernetes handles all interaction with the Kubernetes API including starting
// and stopping tasks, monitoring their status, and fetching logs.
package kubernetes

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/internal/agent"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	// Used to load all auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

type podMetadata struct {
	podName     string
	containerID string
}

// High lever overview of the actors within the kubernetes package:
//
//   pods
//     +- pod(s): manages pod lifecycle. One per container in a task.
//        +- podLogStreamer: stream logs for a specific pod.
//     +- informer: sends updates about pod states
//     +- events: sends updates about kubernetes events.
//     +- requestQueue: queues requests to create / delete kubernetes resources.
//        +- requestProcessingWorkers: processes request to create / delete kubernetes resources.
type pods struct {
	cluster                  *actor.Ref
	namespace                string
	masterServiceName        string
	leaveKubernetesResources bool

	clientSet  *k8sClient.Clientset
	masterIP   string
	masterPort int32

	informer                *actor.Ref
	eventListener           *actor.Ref
	resourceRequestQueue    *actor.Ref
	podNameToPodHandler     map[string]*actor.Ref
	containerIDToPodHandler map[string]*actor.Ref
	podHandlerToMetadata    map[*actor.Ref]podMetadata

	podInterface       typedV1.PodInterface
	configMapInterface typedV1.ConfigMapInterface
}

// Initialize creates a new global agent actor.
func Initialize(
	s *actor.System,
	e *echo.Echo,
	c *actor.Ref,
	namespace string,
	masterServiceName string,
	leaveKubernetesResources bool,
) *actor.Ref {
	podsActor, ok := s.ActorOf(actor.Addr("pods"), &pods{
		cluster:                  c,
		namespace:                namespace,
		masterServiceName:        masterServiceName,
		podNameToPodHandler:      make(map[string]*actor.Ref),
		containerIDToPodHandler:  make(map[string]*actor.Ref),
		podHandlerToMetadata:     make(map[*actor.Ref]podMetadata),
		leaveKubernetesResources: leaveKubernetesResources,
	})
	check.Panic(check.True(ok, "pods address already taken"))

	// We re-use the agents endpoint for the default resource provider.
	e.Any("/agents", api.Route(s, podsActor))
	return podsActor
}

func (p *pods) Receive(ctx *actor.Context) error {
	switch msg := ctx.Message().(type) {
	case actor.PreStart:
		if err := p.startClientSet(ctx); err != nil {
			return err
		}
		if err := p.getMasterIPAndPort(ctx); err != nil {
			return err
		}
		p.startResourceRequestQueue(ctx)
		if err := p.deleteExistingKubernetesResources(ctx); err != nil {
			return err
		}
		p.startPodInformer(ctx)
		p.startEventListener(ctx)

	case sproto.StartPod:
		if err := p.receiveStartPod(ctx, msg); err != nil {
			return err
		}

	case podStatusUpdate:
		p.receivePodStatusUpdate(ctx, msg)

	case podEventUpdate:
		p.receivePodEventUpdate(ctx, msg)

	case sproto.KillTaskContainer:
		p.receiveKillContainer(ctx, msg)

	case resourceDeletionFailed:
		if msg.err != nil {
			ctx.Log().WithError(msg.err).Error("error deleting leftover kubernetes resource")
		}

	case actor.ChildStopped:
		if err := p.cleanUpPodHandler(ctx, msg.Child); err != nil {
			return err
		}

	case actor.ChildFailed:
		switch msg.Child {
		case p.informer:
			return errors.Errorf("pod informer failed")
		case p.eventListener:
			return errors.Errorf("event listener failed")
		case p.resourceRequestQueue:
			return errors.Errorf("resource request actor failed")
		}

		if err := p.cleanUpPodHandler(ctx, msg.Child); err != nil {
			return err
		}

	case echo.Context:
		p.handleAPIRequest(ctx, msg)

	case *apiv1.GetAgentsRequest:
		p.handleGetAgentsRequest(ctx)

	default:
		ctx.Log().Errorf("unexpected message %T", msg)
		return actor.ErrUnexpectedMessage(ctx)
	}
	return nil
}

func (p *pods) startClientSet(ctx *actor.Context) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "error building kubernetes config")
	}

	p.clientSet, err = k8sClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize kubernetes clientSet")
	}

	p.podInterface = p.clientSet.CoreV1().Pods(p.namespace)
	p.configMapInterface = p.clientSet.CoreV1().ConfigMaps(p.namespace)

	ctx.Log().Infof("kubernetes clientSet initialized")
	return nil
}

func (p *pods) getMasterIPAndPort(ctx *actor.Context) error {
	masterService, err := p.clientSet.CoreV1().Services(p.namespace).Get(
		p.masterServiceName, metaV1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get master service")
	}

	p.masterIP = masterService.Spec.ClusterIP
	p.masterPort = masterService.Spec.Ports[0].Port
	ctx.Log().Infof("master URL set to %s:%d", p.masterIP, p.masterPort)
	return nil
}

func (p *pods) deleteExistingKubernetesResources(ctx *actor.Context) error {
	listOptions := metaV1.ListOptions{LabelSelector: determinedLabel}

	configMaps, err := p.configMapInterface.List(listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing config maps")
	}
	for _, configMap := range configMaps.Items {
		if configMap.Namespace != p.namespace {
			continue
		}

		ctx.Tell(p.resourceRequestQueue, deleteKubernetesResources{
			handler: ctx.Self(), configMapName: configMap.Name})
	}

	pods, err := p.podInterface.List(listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing pod")
	}
	for _, pod := range pods.Items {
		if pod.Namespace != p.namespace {
			continue
		}

		ctx.Tell(p.resourceRequestQueue, deleteKubernetesResources{
			handler: ctx.Self(), podName: pod.Name})
	}

	return nil
}

func (p *pods) startPodInformer(ctx *actor.Context) {
	p.informer, _ = ctx.ActorOf("pod-informer", newInformer(p.podInterface, p.namespace, ctx.Self()))
}

func (p *pods) startEventListener(ctx *actor.Context) {
	p.eventListener, _ = ctx.ActorOf(
		"event-listener", newEventListener(p.clientSet, p.namespace, ctx.Self()))
}

func (p *pods) startResourceRequestQueue(ctx *actor.Context) {
	p.resourceRequestQueue, _ = ctx.ActorOf(
		"kubernetes-resource-request-queue",
		newRequestQueue(p.podInterface, p.configMapInterface),
	)
}

func (p *pods) receiveStartPod(ctx *actor.Context, msg sproto.StartPod) error {
	newPodHandler := newPod(
		msg, p.cluster, msg.Spec.ClusterID, p.clientSet, p.namespace, p.masterIP, p.masterPort,
		p.podInterface, p.configMapInterface, p.resourceRequestQueue, p.leaveKubernetesResources,
	)
	ref, ok := ctx.ActorOf(fmt.Sprintf("pod-%s", msg.Spec.ContainerID), newPodHandler)
	if !ok {
		return errors.Errorf("pod actor %s already exists", ref.Address().String())
	}

	ctx.Log().WithField("pod", newPodHandler.podName).WithField(
		"handler", ref.Address()).Infof("registering pod handler")

	if _, alreadyExists := p.podNameToPodHandler[newPodHandler.podName]; alreadyExists {
		return errors.Errorf(
			"attempting to register same pod name: %s multiple times", newPodHandler.podName)
	}

	p.podNameToPodHandler[newPodHandler.podName] = ref
	p.containerIDToPodHandler[msg.Spec.ContainerID] = ref
	p.podHandlerToMetadata[ref] = podMetadata{
		podName:     newPodHandler.podName,
		containerID: msg.Spec.ContainerID,
	}

	return nil
}

func (p *pods) receivePodStatusUpdate(ctx *actor.Context, msg podStatusUpdate) {
	ref, ok := p.podNameToPodHandler[msg.updatedPod.Name]
	if !ok {
		ctx.Log().WithField("pod-name", msg.updatedPod.Name).Warn(
			"received pod status update for un-registered pod")
		return
	}

	ctx.Tell(ref, msg)
}

func (p *pods) receivePodEventUpdate(ctx *actor.Context, msg podEventUpdate) {
	ref, ok := p.podNameToPodHandler[msg.event.InvolvedObject.Name]
	if !ok {
		// We log at the debug level because we are unable to filter
		// pods based on their labels the way we do with pod status updates.
		ctx.Log().WithField("pod-name", msg.event.InvolvedObject.Name).Debug(
			"received pod event for an un-registered pod")
		return
	}

	ctx.Tell(ref, msg)
}

func (p *pods) receiveKillContainer(ctx *actor.Context, msg sproto.KillTaskContainer) {
	ref, ok := p.containerIDToPodHandler[string(msg.ContainerID)]
	if !ok {
		// For multi-pod tasks, when the the chief pod exits,
		// the scheduler will request to terminate pods all other pods
		// that have notified the scheduler that they have exited.
		ctx.Log().WithField("container-id", msg.ContainerID).Info(
			"received stop pod command for unregistered container id")
		return
	}

	ctx.Tell(ref, msg)
}

func (p *pods) cleanUpPodHandler(ctx *actor.Context, podHandler *actor.Ref) error {
	podInfo, ok := p.podHandlerToMetadata[podHandler]
	if !ok {
		return errors.Errorf("unknown pod handler being deleted %s", podHandler.Address())
	}

	ctx.Log().WithField("pod", podInfo.podName).WithField(
		"handler", podHandler.Address()).Infof("de-registering pod handler")
	delete(p.podNameToPodHandler, podInfo.podName)
	delete(p.containerIDToPodHandler, podInfo.containerID)
	delete(p.podHandlerToMetadata, podHandler)

	return nil
}

func (p *pods) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		ctx.Respond(apiCtx.JSON(http.StatusOK, p.summarize(ctx)))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (p *pods) handleGetAgentsRequest(ctx *actor.Context) {
	summaries := p.summarize(ctx)
	response := &apiv1.GetAgentsResponse{}

	for _, summary := range summaries {
		response.Agents = append(response.Agents, agent.ToProtoAgent(summary))
	}
	ctx.Respond(response)
}

// summarize will return all nodes currently in the k8 cluster that have GPUs as agents.
// It will map currently running Determined pods to the slots on these Nodes, marking all other
// slots as Free, even if they are being used by other k8 pods.
func (p *pods) summarize(ctx *actor.Context) map[string]agent.AgentSummary {
	podHandlers := make([]*actor.Ref, 0, len(p.podNameToPodHandler))
	for _, podHandler := range p.podNameToPodHandler {
		podHandlers = append(podHandlers, podHandler)
	}
	results := ctx.AskAll(getPodNodeInfo{}, podHandlers...).GetAll()

	// Separate pods by nodes.
	podByNode := make(map[string][]podNodeInfo)
	for _, result := range results {
		info := result.(podNodeInfo)
		if len(info.nodeName) == 0 {
			// If a pod doesn't have a nodeName it means it has not yet
			// been allocated to a node.
			continue
		}
		podByNode[info.nodeName] = append(podByNode[info.nodeName], info)
	}

	nodes, err := p.clientSet.CoreV1().Nodes().List(metaV1.ListOptions{})
	if err != nil {
		ctx.Log().WithError(err).Error("error listing nodes")
		return nil
	}

	summary := make(map[string]agent.AgentSummary)
	for _, node := range nodes.Items {
		gpuResources := node.Status.Capacity["nvidia.com/gpu"]
		numSlots := gpuResources.Value()
		if numSlots < 1 {
			continue
		}

		slotsSummary := make(agent.SlotsSummary)
		curSlot := 0
		for _, podInfo := range podByNode[node.Name] {
			for i := 0; i < podInfo.numGPUs; i++ {
				if curSlot >= int(numSlots) {
					ctx.Log().Warnf("too many pods mapping to node %s", node.Name)
					continue
				}

				slotsSummary[strconv.Itoa(curSlot)] = agent.SlotSummary{
					ID:        strconv.Itoa(i),
					Device:    device.Device{Type: device.GPU},
					Enabled:   true,
					Container: podInfo.container,
				}
				curSlot++
			}
		}

		for i := curSlot; i < int(numSlots); i++ {
			slotsSummary[strconv.Itoa(i)] = agent.SlotSummary{
				ID:      strconv.Itoa(i),
				Device:  device.Device{Type: device.GPU},
				Enabled: true,
			}
		}

		summary[node.Name] = agent.AgentSummary{
			ID:             node.Name,
			RegisteredTime: node.ObjectMeta.CreationTimestamp.Time,
			Slots:          slotsSummary,
			NumContainers:  len(podByNode[node.Name]),
		}
	}

	return summary
}
