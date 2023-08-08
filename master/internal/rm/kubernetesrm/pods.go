package kubernetesrm

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	k8sV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/actor"
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
	resourcePoolConfigs      []config.ResourcePoolConfig
	baseContainerDefaults    *model.TaskContainerDefaultsConfig
	credsDir                 string

	clientSet        *k8sClient.Clientset
	masterIP         string
	masterPort       int32
	masterTLSConfig  model.TLSClientConfig
	loggingTLSConfig model.TLSClientConfig
	loggingConfig    model.LoggingConfig

	resourceRequestQueue         *requestQueue
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

	mu                 sync.RWMutex
	summarizeCacheLock sync.RWMutex
	summarizeCache     summarizeResult
	summarizeCacheTime time.Time

	handleResourceError func(ctx *actor.Context) errorCallbackFunc

	syslog *logrus.Entry
}

type summarizeResult struct {
	summary map[string]model.AgentSummary
	err     error
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
	resourcePoolConfigs []config.ResourcePoolConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	credsDir string,
	masterIP string,
	masterPort int32,
) *actor.Ref {
	loggingTLSConfig := masterTLSConfig
	if loggingConfig.ElasticLoggingConfig != nil {
		loggingTLSConfig = loggingConfig.ElasticLoggingConfig.Security.TLS
	}
	p := &pods{
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
		resourcePoolConfigs:          resourcePoolConfigs,
		baseContainerDefaults:        taskContainerDefaults,
		credsDir:                     credsDir,
		masterIP:                     masterIP,
		masterPort:                   masterPort,
		currentNodes:                 make(map[string]*k8sV1.Node),
		nodeToSystemResourceRequests: make(map[string]int64),
		podInterfaces:                make(map[string]typedV1.PodInterface),
		configMapInterfaces:          make(map[string]typedV1.ConfigMapInterface),
		syslog:                       logrus.WithField("pod-name", namespace),
	}
	p.handleResourceError = func(ctx *actor.Context) errorCallbackFunc {
		return func(err error) {
			p.resourceErrorCallback(ctx, err)
		}
	}
	podsActor, ok := s.ActorOf(actor.Addr("pods"), p)
	check.Panic(check.True(ok, "pods address already taken"))
	s.Ask(podsActor, actor.Ping{}).Get()

	err := p.startPodInformer(s)
	if err != nil {
		panic(err)
	}

	err = p.startNodeInformer()
	if err != nil {
		panic(err)
	}

	err = p.startEventListeners(s)
	if err != nil {
		panic(err)
	}

	err = p.startPreemptionListeners(s)
	if err != nil {
		panic(err)
	}

	return podsActor
}

func (p *pods) Receive(ctx *actor.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

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

	case actor.PostStop:

	case StartTaskPod:
		if err := p.receiveStartTaskPod(ctx, msg); err != nil {
			return err
		}

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

	case actor.ChildStopped:
		if err := p.cleanUpPodHandler(ctx, msg.Child); err != nil {
			return err
		}

	case actor.ChildFailed:
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

func (p *pods) resourceErrorCallback(ctx *actor.Context, err error) {
	switch err := err.(type) {
	case resourceDeletionFailed:
		ctx.Log().WithError(err).Error("error deleting leftover kubernetes resource")
	default:
		panic(fmt.Sprintf("unexpected message %T", err))
	}
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
	server, err := os.ReadFile(filepath.Join(credsDir, "server"))
	if err != nil {
		return nil, err
	}

	server = bytes.Trim(server, " \t\r\n")

	tokenFile := filepath.Join(credsDir, "token")
	//nolint:gosec // Yes, we intend to read from this file specified in the config.
	token, err := os.ReadFile(tokenFile)
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

	for _, ns := range append(maps.Keys(p.namespaceToPoolName), p.namespace) {
		p.podInterfaces[ns] = p.clientSet.CoreV1().Pods(ns)
		p.configMapInterfaces[ns] = p.clientSet.CoreV1().ConfigMaps(ns)
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

	pods, err := p.listPodsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing pods checking if they can be restored")
	}

	configMaps, err := p.listConfigMapsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing config maps checking if they can be restored")
	}
	existingConfigMaps := make(set.Set[string])
	for _, cm := range configMaps.Items {
		if _, ok := p.namespaceToPoolName[cm.Namespace]; !ok {
			continue
		}
		existingConfigMaps.Insert(cm.Name)
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
					if !existingConfigMaps.Contains(pod.Name) {
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
		resp, err := p.reattachPod(ctx, msg.allocationID, resourcePool, containerID,
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
	allocationID model.AllocationID,
	resourcePool string,
	containerID string,
	pod *k8sV1.Pod,
	ports []int,
	slots int,
	logContext logger.Context,
) (reattachPodResponse, error) {
	startMsg := StartTaskPod{
		AllocationID: allocationID,
		Spec: tasks.TaskSpec{
			ContainerID: containerID,
		},
		Slots:        slots,
		ResourcePool: resourcePool,
		LogContext:   logContext,
	}

	newPodHandler := newPod(
		startMsg,
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

	// Calls podStatusCallback for any missed updates between master going up
	// and the pod being reattached.
	updated, err := p.podInterfaces[pod.Namespace].Get(context.TODO(), pod.Name, metaV1.GetOptions{})
	if err != nil {
		return reattachPodResponse{}, errors.Wrap(err, "error getting pod status update in restore")
	}
	p.podStatusCallback(ctx.Self().System(), watch.Event{Object: updated})

	return reattachPodResponse{containerID: containerID, started: started}, nil
}

func (p *pods) deleteKubernetesResources(
	ctx *actor.Context, pods *k8sV1.PodList, configMaps *k8sV1.ConfigMapList,
) {
	for _, pod := range pods.Items {
		p.resourceRequestQueue.deleteKubernetesResources(
			p.handleResourceError(ctx), pod.Namespace, pod.Name, "",
		)
	}

	for _, configMap := range configMaps.Items {
		p.resourceRequestQueue.deleteKubernetesResources(
			p.handleResourceError(ctx), configMap.Namespace, "", configMap.Name,
		)
	}
}

func (p *pods) deleteDoomedKubernetesResources(ctx *actor.Context) error {
	var openAllocations []model.Allocation
	if err := db.Bun().NewSelect().Model(&openAllocations).
		Where("end_time IS NULL").
		Scan(context.TODO()); err != nil {
		return errors.Wrap(err, "error querying the database for open allocations")
	}
	openAllocationIDs := make(set.Set[model.AllocationID])
	for _, alloc := range openAllocations {
		openAllocationIDs.Insert(alloc.AllocationID)
	}

	listOptions := metaV1.ListOptions{LabelSelector: determinedLabel}
	pods, err := p.listPodsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing pods")
	}
	toKillPods := &k8sV1.PodList{}
	savedPodNames := make(set.Set[string])
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

		if !openAllocationIDs.Contains(model.AllocationID(pod.Labels[determinedLabel])) {
			ctx.Log().Warnf("deleting pod '%s', did not find open allocation '%s'",
				pod.Name, pod.Labels[determinedLabel])
			toKillPods.Items = append(toKillPods.Items, pod)
			continue
		}
		savedPodNames.Insert(pod.Name)
	}

	configMaps, err := p.listConfigMapsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing config maps")
	}
	toKillConfigMaps := &k8sV1.ConfigMapList{}
	for _, cm := range configMaps.Items {
		if _, ok := p.namespaceToPoolName[cm.Namespace]; !ok {
			continue
		}

		if savedPodNames.Contains(cm.Name) { // PodName is same as config map name.
			continue
		}

		ctx.Log().Debugf("Deleting config map '%s' did not find a matching pod that will be restored",
			cm.Name)
		toKillConfigMaps.Items = append(toKillConfigMaps.Items, cm)
	}

	p.deleteKubernetesResources(ctx, toKillPods, toKillConfigMaps)
	return nil
}

func (p *pods) startPodInformer(s *actor.System) error {
	for namespace := range p.namespaceToPoolName {
		i, err := newPodInformer(
			context.TODO(),
			determinedLabel,
			"pod",
			namespace,
			p.podInterfaces[namespace],
			func(event watch.Event) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.podStatusCallback(s, event)
			},
		)
		if err != nil {
			return err
		}

		go i.run(context.TODO())
	}
	return nil
}

func (p *pods) startNodeInformer() error {
	i, err := newNodeInformer(
		context.TODO(),
		p.clientSet.CoreV1().Nodes(),
		func(event watch.Event) {
			p.mu.Lock()
			defer p.mu.Unlock()
			p.nodeStatusCallback(event)
		})
	if err != nil {
		return err
	}

	go i.run(context.TODO())
	return nil
}

func (p *pods) startEventListeners(s *actor.System) error {
	for namespace := range p.namespaceToPoolName {
		l, err := newEventInformer(
			context.TODO(),
			p.clientSet.CoreV1().Events(namespace),
			namespace,
			func(event watch.Event) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.eventStatusCallback(s, event)
			})
		if err != nil {
			return err
		}
		go l.run(context.TODO())
	}
	return nil
}

func (p *pods) startPreemptionListeners(s *actor.System) error {
	for namespace := range p.namespaceToPoolName {
		l, err := newPodInformer(
			context.TODO(),
			determinedPreemptionLabel,
			"preemption",
			namespace,
			p.clientSet.CoreV1().Pods(namespace),
			func(event watch.Event) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.preemptionCallback(s, event)
			})
		if err != nil {
			return err
		}
		go l.run(context.TODO())
	}
	return nil
}

func (p *pods) startResourceRequestQueue(ctx *actor.Context) {
	p.resourceRequestQueue = startRequestQueue(p.podInterfaces, p.configMapInterfaces)
}

func (p *pods) receiveStartTaskPod(ctx *actor.Context, msg StartTaskPod) error {
	newPodHandler := newPod(
		msg,
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

func (p *pods) podStatusCallback(s *actor.System, event watch.Event) {
	pod, ok := event.Object.(*k8sV1.Pod)
	if !ok {
		p.syslog.Warnf("error converting event of type %T to *k8sV1.Pod: %+v", event, event)
		return
	}
	p.syslog.Debugf("informer got new pod event for pod %s: %s ", pod.Name, event.Type)

	ref, ok := p.podNameToPodHandler[pod.Name]
	if !ok {
		p.syslog.Warn("received pod status update for un-registered pod")
		return
	}

	s.Tell(ref, podStatusUpdate{pod})

	if containerID, ok := p.podNameToContainerID[pod.Name]; ok {
		if state, ok := p.containerIDToSchedulingState[containerID]; ok {
			currState := sproto.SchedulingStateQueued
			if pod.Status.Phase == "Running" {
				currState = sproto.SchedulingStateScheduled
			}
			if currState != state {
				p.containerIDToSchedulingState[containerID] = currState
				s.Tell(p.cluster, sproto.UpdatePodStatus{
					ContainerID: containerID,
					State:       currState,
				})
			}
		}
	}
}

func (p *pods) nodeStatusCallback(event watch.Event) {
	node, ok := event.Object.(*k8sV1.Node)
	if !ok {
		p.syslog.Warnf("error converting event of type %T to *k8sV1.Node: %+v", event, event)
		return
	}

	p.syslog.Debugf(`informer got new node event for node '%s': %s %s`,
		node.Name, event.Type, node.Status.Phase)

	switch event.Type {
	case watch.Added:
		p.currentNodes[node.Name] = node
	case watch.Modified:
		p.currentNodes[node.Name] = node
	case watch.Deleted:
		delete(p.currentNodes, node.Name)
	default:
	}
}

func (p *pods) eventStatusCallback(s *actor.System, event watch.Event) {
	newEvent, ok := event.Object.(*k8sV1.Event)
	if !ok {
		p.syslog.Warnf("error converting object type %T to *k8sV1.Event: %+v", event, event)
		return
	}

	p.syslog.Debugf("listener got new event: %s", newEvent.Message)
	ref, ok := p.podNameToPodHandler[newEvent.InvolvedObject.Name]
	if !ok {
		// We log at the debug level because we are unable to filter
		// pods based on their labels the way we do with pod status updates.
		p.syslog.Debug("received pod event for an un-registered pod")
		return
	}

	s.Tell(ref, podEventUpdate{newEvent})
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

func (p *pods) preemptionCallback(s *actor.System, event watch.Event) {
	pod, ok := event.Object.(*k8sV1.Pod)
	if !ok {
		p.syslog.Warnf("error converting event of type %T to *k8sV1.Pod: %+v", event, event)
		return
	}
	p.syslog.Debugf("informer got new preemption event for pod %s ", pod.Name)

	ref, ok := p.podNameToPodHandler[pod.Name]
	if !ok {
		p.syslog.Debug("received preemption command for unregistered pod")
		return
	}
	s.Tell(ref, PreemptTaskPod{PodName: pod.Name})
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
		summaries := p.summarizeClusterByNodes(ctx)
		_, nodesToPools := p.getNodeResourcePoolMapping(summaries)
		for nodeName, summary := range summaries {
			summary.ResourcePool = nodesToPools[summary.ID]
			summaries[nodeName] = summary
		}
		ctx.Respond(apiCtx.JSON(http.StatusOK, summaries))
	default:
		ctx.Respond(echo.ErrMethodNotAllowed)
	}
}

func (p *pods) handleGetAgentsRequest(ctx *actor.Context) {
	nodeSummaries := p.summarizeClusterByNodes(ctx)
	_, nodesToPools := p.getNodeResourcePoolMapping(nodeSummaries)

	response := &apiv1.GetAgentsResponse{}
	for _, summary := range nodeSummaries {
		summary.ResourcePool = nodesToPools[summary.ID]
		response.Agents = append(response.Agents, summary.ToProto())
	}
	ctx.Respond(response)
}

// summarize describes pods' available resources. When there's exactly one resource pool, it uses
// the whole cluster's info. Otherwise, it matches nodes to resource pools using taints and
// tolerations to derive that info. This may be cached, so don't use this for decisions
// that require up-to-date information.
func (p *pods) summarize(ctx *actor.Context) (map[string]model.AgentSummary, error) {
	p.summarizeCacheLock.Lock()
	defer p.summarizeCacheLock.Unlock()

	if time.Since(p.summarizeCacheTime) > 5*time.Second {
		summary, err := p.computeSummary(ctx)
		p.summarizeCacheTime = time.Now()
		p.summarizeCache = summarizeResult{
			summary: summary,
			err:     err,
		}
	}

	return p.summarizeCache.summary, p.summarizeCache.err
}

// Get the mapping of many-to-many relationship between nodes and resource pools.
func (p *pods) getNodeResourcePoolMapping(nodeSummaries map[string]model.AgentSummary) (
	map[string][]*k8sV1.Node, map[string][]string,
) {
	poolTaskContainerDefaults := extractTCDs(p.resourcePoolConfigs)

	// Nvidia automatically taints nodes, so we should tolerate that when users don't customize
	// their resource pool config.
	defaultTolerations := []k8sV1.Toleration{{
		Key:      ResourceTypeNvidia,
		Value:    "present",
		Operator: k8sV1.TolerationOpEqual,
	}}
	cpuTolerations, gpuTolerations := extractTolerations(p.baseContainerDefaults)
	poolsToNodes := make(map[string][]*k8sV1.Node, len(p.namespaceToPoolName))
	nodesToPools := make(map[string][]string, len(p.namespaceToPoolName))

	for _, node := range p.currentNodes {
		_, slotType := extractSlotInfo(nodeSummaries[node.Name])

		for poolName, tcd := range poolTaskContainerDefaults {
			var poolTolerations []k8sV1.Toleration

			// If they're using the default RP config, use the default tolerations.
			if len(p.resourcePoolConfigs) <= 1 &&
				(tcd == nil || (tcd.CPUPodSpec == nil && tcd.GPUPodSpec == nil)) {
				if slotType == device.CUDA {
					//nolint:gocritic
					poolTolerations = append(defaultTolerations, gpuTolerations...)
				} else if slotType == device.CPU {
					//nolint:gocritic
					poolTolerations = append(defaultTolerations, cpuTolerations...)
				}
			} else if tcd != nil {
				// Decide which poolTolerations to use based on slot device type
				if slotType == device.CUDA && tcd.GPUPodSpec != nil {
					//nolint:gocritic
					poolTolerations = append(tcd.GPUPodSpec.Spec.Tolerations, gpuTolerations...)
				} else if tcd.CPUPodSpec != nil {
					//nolint:gocritic
					poolTolerations = append(tcd.CPUPodSpec.Spec.Tolerations, cpuTolerations...)
				}
			}

			// If all of a node's taints are tolerated by a pool, that node belongs to the pool.
			if allTaintsTolerated(node.Spec.Taints, poolTolerations) {
				poolsToNodes[poolName] = append(poolsToNodes[poolName], node)
				nodesToPools[node.Name] = append(nodesToPools[node.Name], poolName)
			}
		}
	}

	return poolsToNodes, nodesToPools
}

func (p *pods) computeSummary(ctx *actor.Context) (map[string]model.AgentSummary, error) {
	nodeSummaries := p.summarizeClusterByNodes(ctx)

	// Build the many-to-many relationship between nodes and resource pools
	poolsToNodes, _ := p.getNodeResourcePoolMapping(nodeSummaries)

	// Build the set of summaries for each resource pool
	containers := p.containersPerResourcePool()
	summaries := make(map[string]model.AgentSummary, len(p.namespaceToPoolName))
	for poolName, nodes := range poolsToNodes {
		slots := model.SlotsSummary{}
		numContainersInPool := containers[poolName]

		// We'll create a number of pseudo-containers in the summary equal to the number of
		// running containers in this pool.
		pseudoContainersAdded := 0

		for _, node := range nodes {
			numSlots, slotType := extractSlotInfo(nodeSummaries[node.Name])

			for j := 0; j < numSlots; j++ {
				id := fmt.Sprintf("%s/%s/%s/%d", poolName, node.Name, string(slotType), j)

				var container *cproto.Container
				if pseudoContainersAdded < numContainersInPool {
					container = &cproto.Container{
						ID:    cproto.ID(id),
						State: "RUNNING",
					}
					pseudoContainersAdded++
				}

				slots[id] = model.SlotSummary{
					ID:        id,
					Device:    device.Device{Type: slotType},
					Enabled:   true,
					Container: container,
				}
			}
		}

		summaries[poolName] = model.AgentSummary{
			ID:             poolName,
			RegisteredTime: p.cluster.RegisteredTime(),
			NumContainers:  numContainersInPool,
			ResourcePool:   []string{poolName},
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
			ResourcePool:   []string{""},
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

func (p *pods) listPodsInAllNamespaces(
	ctx context.Context, opts metaV1.ListOptions,
) (*k8sV1.PodList, error) {
	res := &k8sV1.PodList{}
	for n, i := range p.podInterfaces {
		pods, err := i.List(ctx, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "error listing pods for namespace %s", n)
		}

		res.Items = append(res.Items, pods.Items...)
	}

	return res, nil
}

func (p *pods) listConfigMapsInAllNamespaces(
	ctx context.Context, opts metaV1.ListOptions,
) (*k8sV1.ConfigMapList, error) {
	res := &k8sV1.ConfigMapList{}
	for n, i := range p.configMapInterfaces {
		cms, err := i.List(ctx, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "error listing config maps for namespace %s", n)
		}

		res.Items = append(res.Items, cms.Items...)
	}

	return res, nil
}

func extractTCDs(resourcePoolConfigs []config.ResourcePoolConfig,
) map[string]*model.TaskContainerDefaultsConfig {
	result := map[string]*model.TaskContainerDefaultsConfig{}

	for _, config := range resourcePoolConfigs {
		result[config.PoolName] = config.TaskContainerDefaults
	}

	return result
}

func taintTolerated(taint k8sV1.Taint, tolerations []k8sV1.Toleration) bool {
	for _, toleration := range tolerations {
		if toleration.ToleratesTaint(&taint) {
			return true
		}
	}

	return false
}

func allTaintsTolerated(taints []k8sV1.Taint, tolerations []k8sV1.Toleration) bool {
	for _, taint := range taints {
		if !taintTolerated(taint, tolerations) {
			return false
		}
	}

	return true
}

func extractSlotInfo(node model.AgentSummary) (numSlots int, devType device.Type) {
	var gpuSlots, cpuSlots int

	for _, slot := range node.Slots {
		if slot.Device.Type == device.CPU {
			cpuSlots++
		} else if slot.Device.Type == device.CUDA {
			gpuSlots++
		}
	}

	if gpuSlots > 0 {
		return gpuSlots, device.CUDA
	}

	return cpuSlots, device.CPU
}

func extractTolerations(tcd *model.TaskContainerDefaultsConfig) (
	cpuTolerations, gpuTolerations []k8sV1.Toleration,
) {
	if tcd != nil {
		if tcd.GPUPodSpec != nil {
			gpuTolerations = tcd.GPUPodSpec.Spec.Tolerations
		}
		if tcd.CPUPodSpec != nil {
			cpuTolerations = tcd.CPUPodSpec.Spec.Tolerations
		}
	}

	return cpuTolerations, gpuTolerations
}
