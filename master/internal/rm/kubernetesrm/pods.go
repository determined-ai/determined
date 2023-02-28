package kubernetesrm

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"gopkg.in/inf.v0"
	k8sV1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/actor/api"
	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

	// Used to load all auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// ResourceTypeNvidia describes the GPU resource type.
const ResourceTypeNvidia = "nvidia.com/gpu"

type podMetadata struct {
	podName     string
	containerID string
}

// High lever overview of the actors within the kubernetes package:
//
//	pods
//	  +- pod(s): manages pod lifecycle. One per container in a task.
//	     +- podLogStreamer: stream logs for a specific pod.
//	  +- informer: sends updates about pod states
//	  +- events: sends updates about kubernetes events.
//	  +- requestQueue: queues requests to create / delete kubernetes resources.
//	     +- requestProcessingWorkers: processes request to create / delete kubernetes resources.
type pods struct {
	cluster                  *actor.Ref
	namespace                string
	namespaceToPoolName      map[string]string
	masterServiceName        string
	leaveKubernetesResources bool
	scheduler                string
	slotType                 device.Type
	slotResourceRequests     config.PodSlotResourceRequests
	fluentConfig             config.FluentConfig
	credsDir                 string

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
	podNameToResourcePool        map[string]string
	containerIDToPodName         map[string]string
	containerIDToSchedulingState map[string]sproto.SchedulingState
	podNameToContainerID         map[string]string
	podHandlerToMetadata         map[*actor.Ref]podMetadata
	nodeToSystemResourceRequests map[string]int64

	currentNodes map[string]*k8sV1.Node

	podInterfaces       map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface
	quotaInterfaces     map[string]typedV1.ResourceQuotaInterface
}

// PodsInfo contains information for pods.
type PodsInfo struct {
	NumAgents      int
	SlotsAvailable int
}

// SummarizeResources summerize pods resource.
type SummarizeResources struct {
	PoolName string
}

type reattachAllocationPods struct {
	numPods      int
	allocationID model.AllocationID
	taskActor    *actor.Ref
	slots        int
	logContext   logger.Context
}

type reattachPodResponse struct {
	containerID string
	started     *sproto.ResourcesStarted
}

// Initialize creates a new global pods actor.
func Initialize(
	s *actor.System,
	e *echo.Echo,
	c *actor.Ref,
	namespace string,
	namespaceToPoolName map[string]string,
	masterServiceName string,
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
	leaveKubernetesResources bool,
	scheduler string,
	slotType device.Type,
	slotResourceRequests config.PodSlotResourceRequests,
	fluentConfig config.FluentConfig,
	credsDir string,
	masterIP string,
	masterPort int32,
) *actor.Ref {
	loggingTLSConfig := masterTLSConfig
	if loggingConfig.ElasticLoggingConfig != nil {
		loggingTLSConfig = loggingConfig.ElasticLoggingConfig.Security.TLS
	}

	podsActor, ok := s.ActorOf(actor.Addr("pods"), &pods{
		cluster:                      c,
		namespace:                    namespace,
		namespaceToPoolName:          namespaceToPoolName,
		masterServiceName:            masterServiceName,
		masterTLSConfig:              masterTLSConfig,
		scheduler:                    scheduler,
		loggingTLSConfig:             loggingTLSConfig,
		loggingConfig:                loggingConfig,
		podNameToPodHandler:          make(map[string]*actor.Ref),
		podNameToResourcePool:        make(map[string]string),
		containerIDToPodName:         make(map[string]string),
		containerIDToSchedulingState: make(map[string]sproto.SchedulingState),
		podNameToContainerID:         make(map[string]string),
		podHandlerToMetadata:         make(map[*actor.Ref]podMetadata),
		leaveKubernetesResources:     leaveKubernetesResources,
		slotType:                     slotType,
		slotResourceRequests:         slotResourceRequests,
		fluentConfig:                 fluentConfig,
		credsDir:                     credsDir,
		masterIP:                     masterIP,
		masterPort:                   masterPort,
		currentNodes:                 make(map[string]*k8sV1.Node),
		nodeToSystemResourceRequests: make(map[string]int64),
		podInterfaces:                make(map[string]typedV1.PodInterface),
		configMapInterfaces:          make(map[string]typedV1.ConfigMapInterface),
		quotaInterfaces:              make(map[string]typedV1.ResourceQuotaInterface),
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

		if !p.leaveKubernetesResources {
			if err := p.deleteDoomedKubernetesResources(ctx); err != nil {
				return err
			}
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

	case reattachAllocationPods:
		if err := p.reattachAllocationPods(ctx, msg); err != nil {
			return err
		}

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

func readClientConfig(credsDir string) (*rest.Config, error) {
	if credsDir == "" {
		// The default in-cluster case.  Internally, k8s.io/client-go/rest is going to look for
		// environment variables:
		//   - KUBERNETES_SERVICE_HOST
		//   - KUBERNETES_SERVICE_PORT
		// and it expects to find files:
		//   - /var/run/secrets/kubernetes.io/serviceaccount/token
		//   - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
		return rest.InClusterConfig()
	}

	// A special case for rapid determined+k8s development: build a rest.Config from a specially
	// packed directory with the same information.  Our tools/scripts/fetch-k8s-creds.sh script can
	// create such a directory, with server, token, and ca.crt files.

	//nolint:gosec // Yes, we intend to read from this file specified in the config.
	server, err := ioutil.ReadFile(filepath.Join(credsDir, "server"))
	if err != nil {
		return nil, err
	}

	server = bytes.Trim(server, " \t\r\n")

	tokenFile := filepath.Join(credsDir, "token")
	//nolint:gosec // Yes, we intend to read from this file specified in the config.
	token, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}

	return &rest.Config{
		Host:            string(server),
		BearerToken:     string(token),
		BearerTokenFile: tokenFile,
		TLSClientConfig: rest.TLSClientConfig{
			CAFile: filepath.Join(credsDir, "ca.crt"),
		},
	}, nil
}

func (p *pods) startClientSet(ctx *actor.Context) error {
	config, err := readClientConfig(p.credsDir)
	if err != nil {
		return errors.Wrap(err, "error building kubernetes config")
	}

	p.clientSet, err = k8sClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize kubernetes clientSet")
	}

	for ns := range p.namespaceToPoolName {
		p.quotaInterfaces[ns] = p.clientSet.CoreV1().ResourceQuotas(ns)
		p.podInterfaces[ns] = p.clientSet.CoreV1().Pods(ns)
		p.configMapInterfaces[ns] = p.clientSet.CoreV1().ConfigMaps(ns)
	}
	for _, ns := range []string{metaV1.NamespaceAll, p.namespace} {
		if _, ok := p.namespaceToPoolName[ns]; !ok {
			p.quotaInterfaces[ns] = p.clientSet.CoreV1().ResourceQuotas(ns)
			p.podInterfaces[ns] = p.clientSet.CoreV1().Pods(ns)
			p.configMapInterfaces[ns] = p.clientSet.CoreV1().ConfigMaps(ns)
		}
	}

	ctx.Log().Infof("kubernetes clientSet initialized")
	return nil
}

func (p *pods) getMasterIPAndPort(ctx *actor.Context) error {
	if p.masterIP != "" && p.masterPort != 0 {
		// Master ip and port were manually configured (probably for development purposes).
		return nil
	}
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
	systemPods, err := p.podInterfaces[p.namespace].List(
		context.TODO(), metaV1.ListOptions{LabelSelector: determinedSystemLabel})
	if err != nil {
		return errors.Wrap(err, "failed to get system pods")
	}

	for _, systemPod := range systemPods.Items {
		for _, container := range systemPod.Spec.Containers {
			p.nodeToSystemResourceRequests[systemPod.Spec.NodeName] += container.Resources.Requests.Cpu().
				MilliValue()
		}
	}
	return nil
}

func (p *pods) reattachAllocationPods(ctx *actor.Context, msg reattachAllocationPods) error {
	listOptions := metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", determinedLabel, msg.allocationID),
	}

	pods, err := p.podInterfaces[metaV1.NamespaceAll].List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing pods checking if they can be restored")
	}

	configMaps, err := p.configMapInterfaces[metaV1.NamespaceAll].List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing config maps checking if they can be restored")
	}
	existingConfigMaps := make(map[string]bool)
	for _, cm := range configMaps.Items {
		if _, ok := p.namespaceToPoolName[cm.Namespace]; !ok {
			continue
		}
		existingConfigMaps[cm.Name] = true
	}

	var containerIDs []string
	var k8sPods []*k8sV1.Pod
	var ports [][]int
	var resourcePool string
	for _, pod := range pods.Items {
		if _, ok := p.namespaceToPoolName[pod.Namespace]; !ok {
			continue
		}

		foundID := false
		foundPool := false
		for _, container := range pod.Spec.Containers {
			for _, env := range container.Env {
				switch env.Name {
				case "DET_CONTAINER_ID":
					if !existingConfigMaps[pod.Name] {
						p.deleteKubernetesResources(ctx, pods, configMaps)
						ctx.Respond(fmt.Errorf("pod missing config map %s", pod.Name))
						return nil
					}

					p := pod
					k8sPods = append(k8sPods, &p)
					containerIDs = append(containerIDs, env.Value)

					var podPorts []int
					for _, p := range container.Ports {
						podPorts = append(podPorts, int(p.ContainerPort))
					}
					ports = append(ports, podPorts)

					foundID = true
				case resourcePoolEnvVar:
					resourcePool = env.Value
					foundPool = true
				}
			}
			if foundID && foundPool {
				break
			}
		}
	}

	if len(k8sPods) != msg.numPods {
		p.deleteKubernetesResources(ctx, pods, configMaps)
		ctx.Respond(fmt.Errorf("not enough pods found for allocation expected %d got %d instead",
			msg.numPods, len(k8sPods)))
		return nil
	}

	var restoreResponses []reattachPodResponse
	for i, containerID := range containerIDs {
		resp, err := p.reattachPod(ctx, msg.taskActor, resourcePool, containerID,
			k8sPods[i], ports[i], msg.slots, msg.logContext)
		if err != nil {
			p.deleteKubernetesResources(ctx, pods, configMaps)
			ctx.Respond(errors.Wrapf(err,
				"error restoring pod with containerID %s", containerID))
			return nil
		}
		restoreResponses = append(restoreResponses, resp)
	}

	ctx.Respond(restoreResponses)
	return nil
}

func (p *pods) reattachPod(
	ctx *actor.Context,
	taskActor *actor.Ref,
	resourcePool string,
	containerID string,
	pod *k8sV1.Pod,
	ports []int,
	slots int,
	logContext logger.Context,
) (reattachPodResponse, error) {
	startMsg := StartTaskPod{
		TaskActor: taskActor,
		Spec: tasks.TaskSpec{
			ContainerID: containerID,
		},
		Slots:        slots,
		ResourcePool: resourcePool,
		LogContext:   logContext,
	}

	newPodHandler := newPod(
		startMsg,
		p.cluster,
		startMsg.Spec.ClusterID,
		p.clientSet,
		pod.Namespace,
		p.masterIP,
		p.masterPort,
		p.masterTLSConfig,
		p.loggingTLSConfig,
		p.loggingConfig,
		p.podInterfaces[pod.Namespace],
		p.configMapInterfaces[pod.Namespace],
		p.resourceRequestQueue,
		p.leaveKubernetesResources,
		p.slotType,
		p.slotResourceRequests,
		p.scheduler,
		p.fluentConfig,
	)

	newPodHandler.restore = true
	newPodHandler.logCtx["pod"] = pod.Name
	newPodHandler.podName = pod.Name
	newPodHandler.configMapName = pod.Name
	newPodHandler.ports = ports

	state, err := getPodState(ctx, pod, newPodHandler.containerNames)
	if err != nil {
		return reattachPodResponse{}, errors.Wrap(err, "error finding pod state to restore")
	}
	// Don't set container state if the state is terminated.
	// This is so that when we send the update message we will go
	// through pod shutdown logic and avoid dropping a duplicate state messages.
	if state != cproto.Terminated {
		newPodHandler.container.State = state
	}

	var started *sproto.ResourcesStarted
	if newPodHandler.container.State == cproto.Running {
		started = ptrs.Ptr(getResourcesStartedForPod(pod, newPodHandler.ports))
	}

	newPodHandler.pod = pod

	ref, ok := ctx.ActorOf(fmt.Sprintf("pod-%s", containerID), newPodHandler)
	if !ok {
		return reattachPodResponse{}, errors.Errorf(
			"pod actor %s already exists", ref.Address().String())
	}

	p.podNameToPodHandler[pod.Name] = ref
	p.podNameToResourcePool[pod.Name] = resourcePool
	p.containerIDToPodName[containerID] = pod.Name
	p.podNameToContainerID[pod.Name] = containerID
	p.containerIDToSchedulingState[containerID] = sproto.SchedulingStateQueued
	p.podHandlerToMetadata[ref] = podMetadata{
		podName:     pod.Name,
		containerID: containerID,
	}

	// Send a podStatusUpdate for any missed updates between master going up
	// and the pod being reattached.
	updated, err := p.podInterfaces[pod.Namespace].Get(context.TODO(), pod.Name, metaV1.GetOptions{})
	if err != nil {
		return reattachPodResponse{}, errors.Wrap(err, "error getting pod status update in restore")
	}
	ctx.Tell(ctx.Self(), podStatusUpdate{updatedPod: updated})

	return reattachPodResponse{containerID: containerID, started: started}, nil
}

func (p *pods) deleteKubernetesResources(
	ctx *actor.Context, pods *k8sV1.PodList, configMaps *k8sV1.ConfigMapList,
) {
	for _, pod := range pods.Items {
		ctx.Tell(p.resourceRequestQueue, deleteKubernetesResources{
			handler: ctx.Self(), namespace: pod.Namespace, podName: pod.Name,
		})
	}

	for _, configMap := range configMaps.Items {
		ctx.Tell(p.resourceRequestQueue, deleteKubernetesResources{
			handler: ctx.Self(), namespace: configMap.Namespace, configMapName: configMap.Name,
		})
	}
}

func (p *pods) deleteDoomedKubernetesResources(ctx *actor.Context) error {
	var openAllocations []model.Allocation
	if err := db.Bun().NewSelect().Model(&openAllocations).
		Where("end_time IS NULL").
		Scan(context.TODO()); err != nil {
		return errors.Wrap(err, "error querying the database for open allocations")
	}
	openAllocationIDs := make(map[model.AllocationID]bool)
	for _, alloc := range openAllocations {
		openAllocationIDs[alloc.AllocationID] = true
	}

	listOptions := metaV1.ListOptions{LabelSelector: determinedLabel}
	pods, err := p.podInterfaces[metaV1.NamespaceAll].List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing pods")
	}
	toKillPods := &k8sV1.PodList{}
	savedPodNames := make(map[string]bool)
	for _, pod := range pods.Items {
		if _, ok := p.namespaceToPoolName[pod.Namespace]; !ok {
			continue
		}

		resourcePool := (func() string {
			for _, c := range pod.Spec.Containers {
				for _, e := range c.Env {
					if e.Name == resourcePoolEnvVar {
						return e.Value
					}
				}
			}
			return ""
		})()

		if resourcePool == "" {
			ctx.Log().Debugf("deleting pod '%s' without environment variable '%s'",
				pod.Name, resourcePoolEnvVar)
			toKillPods.Items = append(toKillPods.Items, pod)
			continue
		}
		if !isReattachEnabledForRP(resourcePool) {
			ctx.Log().Debugf("deleting pod '%s' in resource pool '%s' since "+
				"agent_reattach_enabled is disabled", pod.Name, resourcePool)
			toKillPods.Items = append(toKillPods.Items, pod)
			continue
		}

		if !openAllocationIDs[model.AllocationID(pod.Labels[determinedLabel])] {
			ctx.Log().Warnf("deleting pod '%s', did not find open allocation '%s'",
				pod.Name, pod.Labels[determinedLabel])
			toKillPods.Items = append(toKillPods.Items, pod)
			continue
		}
		savedPodNames[pod.Name] = true
	}

	configMaps, err := p.configMapInterfaces[metaV1.NamespaceAll].List(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing config maps")
	}
	toKillConfigMaps := &k8sV1.ConfigMapList{}
	for _, cm := range configMaps.Items {
		if _, ok := p.namespaceToPoolName[cm.Namespace]; !ok {
			continue
		}

		if savedPodNames[cm.Name] { // PodName is same as config map name.
			continue
		}

		ctx.Log().Debugf("Deleting config map '%s' did not find a matching pod that will be restored",
			cm.Name)
		toKillConfigMaps.Items = append(toKillConfigMaps.Items, cm)
	}

	p.deleteKubernetesResources(ctx, toKillPods, toKillConfigMaps)
	return nil
}

func (p *pods) startPodInformer(ctx *actor.Context) {
	p.informer, _ = ctx.ActorOf(
		"pod-informer",
		newInformer(p.podInterfaces[metaV1.NamespaceAll], ctx.Self()),
	)
}

func (p *pods) startNodeInformer(ctx *actor.Context) {
	p.nodeInformer, _ = ctx.ActorOf("node-informer", newNodeInformer(p.clientSet, ctx.Self()))
}

func (p *pods) startEventListener(ctx *actor.Context) {
	p.eventListener, _ = ctx.ActorOf(
		"event-listener", newEventListener(p.clientSet, ctx.Self(),
			set.FromKeys(p.namespaceToPoolName)))
}

func (p *pods) startPreemptionListener(ctx *actor.Context) {
	p.preemptionListener, _ = ctx.ActorOf(
		"preemption-listener", newPreemptionListener(p.clientSet, ctx.Self(),
			set.FromKeys(p.namespaceToPoolName)))
}

func (p *pods) startResourceRequestQueue(ctx *actor.Context) {
	p.resourceRequestQueue, _ = ctx.ActorOf(
		"kubernetes-resource-request-queue",
		newRequestQueue(p.podInterfaces, p.configMapInterfaces),
	)
}

func (p *pods) receiveStartTaskPod(ctx *actor.Context, msg StartTaskPod) error {
	newPodHandler := newPod(
		msg,
		p.cluster,
		msg.Spec.ClusterID,
		p.clientSet,
		msg.Namespace,
		p.masterIP,
		p.masterPort,
		p.masterTLSConfig,
		p.loggingTLSConfig,
		p.loggingConfig,
		p.podInterfaces[msg.Namespace],
		p.configMapInterfaces[msg.Namespace],
		p.resourceRequestQueue,
		p.leaveKubernetesResources,
		p.slotType,
		p.slotResourceRequests,
		p.scheduler,
		p.fluentConfig,
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
	p.podNameToResourcePool[newPodHandler.podName] = msg.ResourcePool
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
	summary, err := p.summarize(ctx)
	if err != nil {
		ctx.Respond(err)
		return
	}

	slots := 0
	if len(msg.PoolName) > 0 {
		slots = numSlots(summary[msg.PoolName].Slots)
	} else {
		for _, pool := range summary {
			slots += numSlots(pool.Slots)
		}
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
	delete(p.podNameToResourcePool, podInfo.podName)
	delete(p.podNameToContainerID, podInfo.podName)
	delete(p.containerIDToPodName, podInfo.containerID)
	delete(p.containerIDToSchedulingState, podInfo.containerID)
	delete(p.podHandlerToMetadata, podHandler)

	return nil
}

func (p *pods) handleAPIRequest(ctx *actor.Context, apiCtx echo.Context) {
	switch apiCtx.Request().Method {
	case echo.GET:
		summary, err := p.summarize(ctx)
		if err != nil {
			ctx.Respond(apiCtx.JSON(http.StatusInternalServerError, err))
		}
		ctx.Respond(apiCtx.JSON(http.StatusOK, summary))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (p *pods) handleGetAgentsRequest(ctx *actor.Context) {
	summaries, err := p.summarize(ctx)
	if err != nil {
		ctx.Respond(err)
		return
	}

	response := &apiv1.GetAgentsResponse{}

	for _, summary := range summaries {
		response.Agents = append(response.Agents, summary.ToProto())
	}
	ctx.Respond(response)
}

// summarize describes pods' available resources. When there's exactly one resource pool and that
// pool has no quotas configured, it uses the whole cluster's info. Otherwise, it uses namespaces'
// quotas to derive that info.
func (p *pods) summarize(ctx *actor.Context) (map[string]model.AgentSummary, error) {
	namespaceToQuota := make(map[string]k8sV1.ResourceQuota)

	// Look up quotas for our resource pools' namespaces.
	for namespace := range p.namespaceToPoolName {
		quotaList, err := p.quotaInterfaces[namespace].List(context.TODO(), metaV1.ListOptions{})
		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, err
		} else if k8serrors.IsNotFound(err) || quotaList == nil {
			continue
		}

		relevantQuotas := cpuAndGpuQuotas(quotaList)
		if len(relevantQuotas) != 1 {
			// TODO: figure out how we want to handle multiple quotas per namespace?
			// When there's multiple conflicting quotas, k8s seems to use the most
			// restrictive of them—i.e. if there's a quota limiting to 100 CPUs and one
			// limiting to 10, only 10 CPUs will be allowed.
			continue
		}

		namespaceToQuota[namespace] = relevantQuotas[0]
	}

	// If there's only one resource pool configured and it doesn't have a quota, summarize using the
	// whole cluster.
	if len(p.namespaceToPoolName) == 1 {
		var namespaceOfPool string
		for namespace := range p.namespaceToPoolName {
			namespaceOfPool = namespace
		}

		// If there's no quota for our only resource pool's namespace
		if _, ok := namespaceToQuota[namespaceOfPool]; !ok {
			return p.summarizeClusterByNodes(ctx), nil
		}
	}

	containers := p.containersPerResourcePool()
	summaries := make(map[string]model.AgentSummary, len(p.namespaceToPoolName))
	for namespace, poolName := range p.namespaceToPoolName {
		slots := model.SlotsSummary{}
		numContainers := containers[poolName]
		var registeredTime time.Time
		if quota, quotaExists := namespaceToQuota[namespace]; quotaExists {
			slots = make(map[string]model.SlotSummary)
			registeredTime = quota.CreationTimestamp.Time

			for resourceName, qty := range quota.Spec.Hard {
				var deviceType device.Type
				switch resourceName {
				case k8sV1.ResourceCPU:
					deviceType = device.CPU
				case ResourceTypeNvidia, "limits." + ResourceTypeNvidia:
					deviceType = device.CUDA
				default:
					// We only care about CPU and GPU quotas for the slots summary
					continue
				}

				// Each CPU and GPU in the quota will be counted as a slot here
				one, decQty := inf.NewDec(1, 0), qty.AsDec()
				for i := inf.NewDec(0, 0); i.Cmp(decQty) < 0; i.Add(i, one) {
					id := fmt.Sprintf("%s/%s/%s", poolName, string(deviceType), i.String())

					var container *cproto.Container
					// Create a number of pseudo-containers in the summary equal to the number of
					// running containers
					if decNumContainers := inf.NewDec(int64(numContainers),
						0); i.Cmp(decNumContainers) < 0 {
						container = &cproto.Container{
							ID:    cproto.ID(id),
							State: "RUNNING",
						}
					}

					slots[id] = model.SlotSummary{
						ID:        id,
						Device:    device.Device{Type: deviceType},
						Enabled:   true,
						Container: container,
					}
				}
			}
		}

		summaries[poolName] = model.AgentSummary{
			ID:             poolName,
			RegisteredTime: registeredTime,
			NumContainers:  numContainers,
			ResourcePool:   poolName,
			Slots:          slots,
		}
	}

	return summaries, nil
}

func (p *pods) summarizeClusterByNodes(ctx *actor.Context) map[string]model.AgentSummary {
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
			resources := node.Status.Allocatable[ResourceTypeNvidia]
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
						ID:          cproto.ID(taskName),
						State:       "RUNNING",
						Devices:     []device.Device{},
						Description: "unknown",
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
				reqs += c.Resources.Requests.Name(ResourceTypeNvidia, resource.DecimalSI).Value()
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

func (p *pods) containersPerResourcePool() map[string]int {
	counts := make(map[string]int, len(p.namespaceToPoolName))
	for _, pool := range p.podNameToResourcePool {
		counts[pool]++
	}
	return counts
}

func numSlots(slots model.SlotsSummary) int {
	slotCountsByType := make(map[device.Type]int)
	for _, slot := range slots {
		slotCountsByType[slot.Device.Type]++
	}

	if slotCountsByType[device.CUDA] > 0 {
		return slotCountsByType[device.CUDA]
	}

	return slotCountsByType[device.CPU]
}

func cpuAndGpuQuotas(quotas *k8sV1.ResourceQuotaList) []k8sV1.ResourceQuota {
	if quotas == nil || len(quotas.Items) == 0 {
		return nil
	}

	result := []k8sV1.ResourceQuota{}
	for _, q := range quotas.Items {
		for resourceName := range q.Spec.Hard {
			switch resourceName {
			case k8sV1.ResourceCPU, ResourceTypeNvidia, "limits." + ResourceTypeNvidia:
				result = append(result, q)
			}
		}
	}

	return result
}
