// Package kubernetes handles all interaction with the Kubernetes API including starting
// and stopping tasks, monitoring their status, and fetching logs.
package kubernetes

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/determined-ai/determined/master/pkg/cproto"

	"github.com/determined-ai/determined/master/internal/sproto"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

	k8sV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
	scheduler                string
	slotType                 device.Type
	slotResourceRequests     PodSlotResourceRequests
	fluentConfig             FluentConfig

	clientSet        *k8sClient.Clientset
	masterIP         string
	masterPort       int32
	masterTLSConfig  model.TLSClientConfig
	loggingTLSConfig model.TLSClientConfig
	loggingConfig    model.LoggingConfig

	informer                     *actor.Ref
	nodeInformer                 *actor.Ref
	eventListener                *actor.Ref
	preemptionListener           *actor.Ref
	resourceRequestQueue         *actor.Ref
	podNameToPodHandler          map[string]*actor.Ref
	containerIDToPodName         map[string]string
	containerIDToSchedulingState map[string]sproto.SchedulingState
	podNameToContainerID         map[string]string
	podHandlerToMetadata         map[*actor.Ref]podMetadata
	nodeToSystemResourceRequests map[string]int64

	currentNodes map[string]*k8sV1.Node

	podInterface       typedV1.PodInterface
	configMapInterface typedV1.ConfigMapInterface
}

// PodsInfo contains information for pods.
type PodsInfo struct {
	NumAgents      int
	SlotsAvailable int
}

// SummarizeResources summerize pods resource.
type SummarizeResources struct{}

// Initialize creates a new global agent actor.
func Initialize(
	s *actor.System,
	e *echo.Echo,
	c *actor.Ref,
	namespace string,
	masterServiceName string,
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
	leaveKubernetesResources bool,
	scheduler string,
	slotType device.Type,
	slotResourceRequests PodSlotResourceRequests,
	fluentConfig FluentConfig,
) *actor.Ref {
	loggingTLSConfig := masterTLSConfig
	if loggingConfig.ElasticLoggingConfig != nil {
		loggingTLSConfig = loggingConfig.ElasticLoggingConfig.Security.TLS
	}

	podsActor, ok := s.ActorOf(actor.Addr("pods"), &pods{
		cluster:                      c,
		namespace:                    namespace,
		masterServiceName:            masterServiceName,
		masterTLSConfig:              masterTLSConfig,
		scheduler:                    scheduler,
		loggingTLSConfig:             loggingTLSConfig,
		loggingConfig:                loggingConfig,
		podNameToPodHandler:          make(map[string]*actor.Ref),
		containerIDToPodName:         make(map[string]string),
		containerIDToSchedulingState: make(map[string]sproto.SchedulingState),
		podNameToContainerID:         make(map[string]string),
		podHandlerToMetadata:         make(map[*actor.Ref]podMetadata),
		leaveKubernetesResources:     leaveKubernetesResources,
		slotType:                     slotType,
		slotResourceRequests:         slotResourceRequests,
		fluentConfig:                 fluentConfig,
		currentNodes:                 make(map[string]*k8sV1.Node),
		nodeToSystemResourceRequests: make(map[string]int64),
	})
	check.Panic(check.True(ok, "pods address already taken"))
	s.Ask(podsActor, actor.Ping{}).Get()

	// We re-use the agents endpoint for the default resource manager.
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
		if err := p.getSystemResourceRequests(ctx); err != nil {
			return err
		}
		p.startResourceRequestQueue(ctx)
		if err := p.deleteExistingKubernetesResources(ctx); err != nil {
			return err
		}
		p.startPodInformer(ctx)
		p.startNodeInformer(ctx)
		p.startEventListener(ctx)
		p.startPreemptionListener(ctx)

	case actor.PostStop:

	case StartTaskPod:
		if err := p.receiveStartTaskPod(ctx, msg); err != nil {
			return err
		}

	case podStatusUpdate:
		p.receivePodStatusUpdate(ctx, msg)

	case nodeStatusUpdate:
		p.receiveNodeStatusUpdate(ctx, msg)

	case podEventUpdate:
		p.receivePodEventUpdate(ctx, msg)

	case PreemptTaskPod:
		p.receivePodPreemption(ctx, msg)

	case ChangePriority:
		p.receivePriorityChange(ctx, msg)

	case ChangePosition:
		p.receivePositionChange(ctx, msg)

	case KillTaskPod:
		p.receiveKillPod(ctx, msg)

	case SummarizeResources:
		p.receiveResourceSummarize(ctx, msg)

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
		case p.nodeInformer:
			return errors.Errorf("node informer failed")
		case p.eventListener:
			return errors.Errorf("event listener failed")
		case p.preemptionListener:
			return errors.Errorf("preemption listener failed")
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
		context.TODO(), p.masterServiceName, metaV1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get master service")
	}

	p.masterIP = masterService.Spec.ClusterIP
	p.masterPort = masterService.Spec.Ports[0].Port
	ctx.Log().Infof("master URL set to %s:%d", p.masterIP, p.masterPort)
	return nil
}

func (p *pods) getSystemResourceRequests(ctx *actor.Context) error {
	systemPods, err := p.podInterface.List(
		context.TODO(), metaV1.ListOptions{LabelSelector: determinedSystemLabel})
	if err != nil {
		return errors.Wrap(err, "failed to get system pods")
	}

	for _, systemPod := range systemPods.Items {
		for _, container := range systemPod.Spec.Containers {
			//nolint:lll // There isn't a great way to break this line that makes it more readable.
			p.nodeToSystemResourceRequests[systemPod.Spec.NodeName] += container.Resources.Requests.Cpu().MilliValue()
		}
	}
	return nil
}

func (p *pods) deleteExistingKubernetesResources(ctx *actor.Context) error {
	listOptions := metaV1.ListOptions{LabelSelector: determinedLabel}

	configMaps, err := p.configMapInterface.List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing config maps")
	}
	for _, configMap := range configMaps.Items {
		if configMap.Namespace != p.namespace {
			continue
		}

		ctx.Tell(p.resourceRequestQueue, deleteKubernetesResources{
			handler: ctx.Self(), configMapName: configMap.Name,
		})
	}

	pods, err := p.podInterface.List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing pod")
	}
	for _, pod := range pods.Items {
		if pod.Namespace != p.namespace {
			continue
		}

		ctx.Tell(p.resourceRequestQueue, deleteKubernetesResources{
			handler: ctx.Self(), podName: pod.Name,
		})
	}

	return nil
}

func (p *pods) startPodInformer(ctx *actor.Context) {
	p.informer, _ = ctx.ActorOf("pod-informer", newInformer(p.podInterface, p.namespace, ctx.Self()))
}

func (p *pods) startNodeInformer(ctx *actor.Context) {
	p.nodeInformer, _ = ctx.ActorOf("node-informer", newNodeInformer(p.clientSet, ctx.Self()))
}

func (p *pods) startEventListener(ctx *actor.Context) {
	p.eventListener, _ = ctx.ActorOf(
		"event-listener", newEventListener(p.clientSet, p.namespace, ctx.Self()))
}

func (p *pods) startPreemptionListener(ctx *actor.Context) {
	p.preemptionListener, _ = ctx.ActorOf(
		"preemption-listener", newPreemptionListener(p.clientSet, p.namespace, ctx.Self()))
}

func (p *pods) startResourceRequestQueue(ctx *actor.Context) {
	p.resourceRequestQueue, _ = ctx.ActorOf(
		"kubernetes-resource-request-queue",
		newRequestQueue(p.podInterface, p.configMapInterface),
	)
}

func (p *pods) receiveStartTaskPod(ctx *actor.Context, msg StartTaskPod) error {
	newPodHandler := newPod(
		msg, p.cluster, msg.Spec.ClusterID, p.clientSet, p.namespace, p.masterIP, p.masterPort,
		p.masterTLSConfig, p.loggingTLSConfig, p.loggingConfig, p.podInterface, p.configMapInterface,
		p.resourceRequestQueue, p.leaveKubernetesResources,
		p.slotType, p.slotResourceRequests, p.scheduler, p.fluentConfig,
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
	p.containerIDToPodName[msg.Spec.ContainerID] = newPodHandler.podName
	p.podNameToContainerID[newPodHandler.podName] = msg.Spec.ContainerID
	p.containerIDToSchedulingState[msg.Spec.ContainerID] = sproto.SchedulingStateQueued
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

	if containerID, ok := p.podNameToContainerID[msg.updatedPod.Name]; ok {
		if state, ok := p.containerIDToSchedulingState[containerID]; ok {
			currState := sproto.SchedulingStateQueued
			if msg.updatedPod.Status.Phase == "Running" {
				currState = sproto.SchedulingStateScheduled
			}
			if currState != state {
				ctx.Tell(p.cluster, sproto.UpdatePodStatus{
					ContainerID: containerID,
					State:       currState,
				})
			}
		}
	}
}

func (p *pods) receiveNodeStatusUpdate(ctx *actor.Context, msg nodeStatusUpdate) {
	if msg.updatedNode != nil {
		p.currentNodes[msg.updatedNode.Name] = msg.updatedNode
	}

	if msg.deletedNode != nil {
		delete(p.currentNodes, msg.deletedNode.Name)
	}
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

func (p *pods) receiveResourceSummarize(ctx *actor.Context, msg SummarizeResources) {
	summary := p.summarize(ctx)
	slots := 0
	for _, node := range summary {
		slots += len(node.Slots)
	}
	ctx.Respond(&PodsInfo{NumAgents: len(summary), SlotsAvailable: slots})
}

func (p *pods) receivePodPreemption(ctx *actor.Context, msg PreemptTaskPod) {
	ref, ok := p.podNameToPodHandler[msg.PodName]
	if !ok {
		ctx.Log().WithField("pod-name", msg.PodName).Debug(
			"received preemption command for unregistered pod")
		return
	}
	ctx.Tell(ref, msg)
}

func (p *pods) verifyPodAndGetRef(ctx *actor.Context, podID string) *actor.Ref {
	podName, ok := p.containerIDToPodName[podID]
	if !ok {
		ctx.Log().WithField("pod-id", podID).Debug(
			"received change priority command for unregistered container id")
		return nil
	}
	ref, ok := p.podNameToPodHandler[podName]
	if !ok {
		ctx.Log().WithField("pod-id", podID).Debug(
			"received change priority command for unregistered container id")
		return nil
	}

	return ref
}

func (p *pods) receivePriorityChange(ctx *actor.Context, msg ChangePriority) {
	ref := p.verifyPodAndGetRef(ctx, msg.PodID.String())
	if ref != nil {
		ctx.Tell(ref, msg)
	}
}

func (p *pods) receivePositionChange(ctx *actor.Context, msg ChangePosition) {
	ref := p.verifyPodAndGetRef(ctx, msg.PodID.String())
	if ref != nil {
		ctx.Tell(ref, msg)
	}
}

func (p *pods) receiveKillPod(ctx *actor.Context, msg KillTaskPod) {
	name, ok := p.containerIDToPodName[string(msg.PodID)]
	if !ok {
		// For multi-pod tasks, when the chief pod exits, the scheduler
		// will request to terminate pods all other pods that have
		// notified the scheduler that they have exited.
		ctx.Log().WithField("pod-id", msg.PodID).Info(
			"received stop pod command for unregistered container id")
		return
	}

	ref, ok := p.podNameToPodHandler[name]
	if !ok {
		ctx.Log().WithField("pod-id", msg.PodID).Info(
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

	name := fmt.Sprintf("%s-priorityclass", podInfo.containerID)
	_, exists := p.clientSet.SchedulingV1().PriorityClasses().Get(
		context.TODO(), name, metaV1.GetOptions{})
	if exists == nil {
		err := p.clientSet.SchedulingV1().PriorityClasses().Delete(
			context.TODO(), name, metaV1.DeleteOptions{})
		if err != nil {
			ctx.Log().Warnf("Deletion of PriorityClass %s failed.", name)
		}
	}

	ctx.Log().WithField("pod", podInfo.podName).WithField(
		"handler", podHandler.Address()).Infof("de-registering pod handler")
	delete(p.podNameToPodHandler, podInfo.podName)
	delete(p.podNameToContainerID, podInfo.podName)
	delete(p.containerIDToPodName, podInfo.containerID)
	delete(p.containerIDToSchedulingState, podInfo.containerID)
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
		response.Agents = append(response.Agents, summary.ToProto())
	}
	ctx.Respond(response)
}

// summarize will return all nodes currently in the k8 cluster that have GPUs as agents.
// It will map currently running Determined pods to the slots on these Nodes, marking all other
// slots as Free, even if they are being used by other k8 pods.
func (p *pods) summarize(ctx *actor.Context) map[string]model.AgentSummary {
	podHandlers := make([]*actor.Ref, 0, len(p.podNameToPodHandler))
	for _, podHandler := range p.podNameToPodHandler {
		podHandlers = append(podHandlers, podHandler)
	}
	results := ctx.AskAll(getPodNodeInfo{}, podHandlers...).GetAll()

	// Separate pods by nodes.
	podByNode := make(map[string][]podNodeInfo, len(results))
	for _, result := range results {
		info := result.(podNodeInfo)
		if len(info.nodeName) == 0 {
			// If a pod doesn't have a nodeName it means it has not yet
			// been allocated to a node.
			continue
		}
		podByNode[info.nodeName] = append(podByNode[info.nodeName], info)
	}

	nodeToTasks, taskSlots := p.getNonDetSlots(p.slotType)

	summary := make(map[string]model.AgentSummary, len(p.currentNodes))
	for _, node := range p.currentNodes {
		var numSlots int64
		var deviceType device.Type
		switch p.slotType {
		case device.CPU:
			resources := node.Status.Allocatable["cpu"]
			milliCPUs := resources.MilliValue() - p.nodeToSystemResourceRequests[node.Name]
			numSlots = int64(float32(milliCPUs) / (1000. * p.slotResourceRequests.CPU))
			deviceType = device.CPU
		case device.ROCM:
			panic("ROCm is not supported on k8s yet")
		case device.CUDA:
			fallthrough
		default:
			resources := node.Status.Allocatable["nvidia.com/gpu"]
			numSlots = resources.Value()
			deviceType = device.CUDA
		}
		if numSlots < 1 {
			continue
		}

		slotsSummary := make(model.SlotsSummary)
		curSlot := 0
		for _, podInfo := range podByNode[node.Name] {
			for i := 0; i < podInfo.numSlots; i++ {
				if curSlot >= int(numSlots) {
					ctx.Log().Warnf("too many pods mapping to node %s", node.Name)
					continue
				}

				slotsSummary[strconv.Itoa(curSlot)] = model.SlotSummary{
					ID:        strconv.Itoa(i),
					Device:    device.Device{Type: deviceType},
					Enabled:   true,
					Container: podInfo.container,
				}
				curSlot++
			}
		}

		for _, taskName := range nodeToTasks[node.Name] {
			for i := int64(0); i < taskSlots[taskName]; i++ {
				if curSlot >= int(numSlots) {
					ctx.Log().Warnf("too many pods mapping to node %s", node.Name)
					continue
				}

				slotsSummary[strconv.Itoa(curSlot)] = model.SlotSummary{
					ID:      strconv.FormatInt(i, 10),
					Device:  device.Device{Type: deviceType},
					Enabled: true,
					Container: &cproto.Container{
						Parent:  actor.Addr(""),
						ID:      cproto.ID(taskName),
						State:   "RUNNING",
						Devices: []device.Device{},
					},
				}
				curSlot++
			}
		}

		for i := curSlot; i < int(numSlots); i++ {
			slotsSummary[strconv.Itoa(i)] = model.SlotSummary{
				ID:      strconv.Itoa(i),
				Device:  device.Device{Type: deviceType},
				Enabled: true,
			}
		}

		var addrs []string
		for _, addr := range node.Status.Addresses {
			addrs = append(addrs, addr.Address)
		}

		summary[node.Name] = model.AgentSummary{
			ID:             node.Name,
			RegisteredTime: node.ObjectMeta.CreationTimestamp.Time,
			Slots:          slotsSummary,
			NumContainers:  len(podByNode[node.Name]) + len(nodeToTasks[node.Name]),
			ResourcePool:   "",
			Addresses:      addrs,
		}
	}

	return summary
}

func (p *pods) getNonDetPods() []k8sV1.Pod {
	var nonDetPods []k8sV1.Pod
	pList, err := p.clientSet.CoreV1().Pods("default").List(context.TODO(), metaV1.ListOptions{})
	if err != nil {
		return nonDetPods
	}
	for _, p := range pList.Items {
		if _, ok := p.Labels["determined"]; !ok {
			if p.Spec.NodeName != "" {
				nonDetPods = append(nonDetPods, p)
			}
		}
	}
	return nonDetPods
}

func (p *pods) getNonDetSlots(deviceType device.Type) (map[string][]string, map[string]int64) {
	nodeToTasks := make(map[string][]string, len(p.currentNodes))
	taskSlots := make(map[string]int64)

	nonDetPods := p.getNonDetPods()
	if len(nonDetPods) == 0 {
		return nodeToTasks, taskSlots
	}
	for _, node := range p.currentNodes {
		nodeToTasks[node.Name] = []string{}
	}

	for _, pod := range nonDetPods {
		if _, ok := nodeToTasks[pod.Spec.NodeName]; !ok {
			continue
		}
		reqs := int64(0)
		for _, c := range pod.Spec.Containers {
			if deviceType == device.CPU {
				reqs += p.getCPUReqs(c)
			} else if deviceType == device.CUDA {
				reqs += c.Resources.Requests.Name("nvidia.com/gpu", resource.DecimalSI).Value()
			}
		}
		if reqs > 0 {
			nodeToTasks[pod.Spec.NodeName] = append(nodeToTasks[pod.Spec.NodeName], pod.Name)
			taskSlots[pod.Name] = reqs
		}
	}
	return nodeToTasks, taskSlots
}

func (p *pods) getCPUReqs(c k8sV1.Container) int64 {
	requested := float32(c.Resources.Requests.Cpu().MilliValue()) /
		(1000. * p.slotResourceRequests.CPU)
	return int64(requested)
}
