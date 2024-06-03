package kubernetesrm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	k8sV1 "k8s.io/api/core/v1"
	k8error "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	k8sClient "k8s.io/client-go/kubernetes"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

	// Used to load all auth plugins.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

// ResourceTypeNvidia describes the GPU resource type.
const ResourceTypeNvidia = "nvidia.com/gpu"

const (
	getAgentsCacheDuration = 15 * time.Second
	summarizeCacheDuration = 5 * time.Second
)

type podMetadata struct {
	podName     string
	containerID string
}

type podStatusUpdateCallback func(sproto.UpdatePodStatus)

// High lever overview of the actors within the kubernetes package:
//
//	pods
//	  +- pod(s): manages pod lifecycle. One per container in a task.
//	     +- podLogStreamer: stream logs for a specific pod.
//	  +- informer: sends updates about pod states
//	  +- events: sends updates about kubernetes events.
//	  +- requestQueue: queues requests to create / delete kubernetes resources.
//	     +- requestProcessingWorkers: processes request to create / delete kubernetes resources.
//
// TODO(DET-10011): Give this literal a more intuitive name.
type pods struct {
	mu sync.RWMutex
	wg waitgroupx.Group

	namespace             string
	namespaceToPoolName   map[string]string
	masterServiceName     string
	scheduler             string
	slotType              device.Type
	slotResourceRequests  config.PodSlotResourceRequests
	resourcePoolConfigs   []config.ResourcePoolConfig
	baseContainerDefaults *model.TaskContainerDefaultsConfig

	kubeconfigPath string

	clientSet        k8sClient.Interface
	detMasterIP      string
	detMasterPort    int32
	masterTLSConfig  model.TLSClientConfig
	loggingTLSConfig model.TLSClientConfig
	loggingConfig    model.LoggingConfig

	resourceRequestQueue         *requestQueue
	podNameToPodHandler          map[string]*pod
	podNameToResourcePool        map[string]string
	containerIDToPodName         map[string]string
	containerIDToSchedulingState map[string]sproto.SchedulingState
	podNameToContainerID         map[string]string
	podHandlerToMetadata         map[*pod]podMetadata
	nodeToSystemResourceRequests map[string]int64

	currentNodes map[string]*k8sV1.Node

	podInterfaces       map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface

	// TODO(RM-236) make one cache and make this code more straightforward.
	summarizeCacheLock sync.RWMutex
	summarizeCache     summarizeResult
	summarizeCacheTime time.Time
	getAgentsCacheLock sync.Mutex
	getAgentsCache     *apiv1.GetAgentsResponse
	getAgentsCacheTime time.Time

	syslog *logrus.Entry

	podStatusUpdateCallback podStatusUpdateCallback
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
	req          *sproto.AllocateRequest
	numPods      int
	allocationID model.AllocationID
	slots        int
	logContext   logger.Context
}

type reattachPodResponse struct {
	containerID string
	started     *sproto.ResourcesStarted
}

type refreshPodStates struct {
	allocationID model.AllocationID
}

// newPodsService creates a new pod service for launching, querying and interacting with k8s pods.
func newPodsService(
	namespace string,
	namespaceToPoolName map[string]string,
	masterServiceName string,
	masterTLSConfig model.TLSClientConfig,
	loggingConfig model.LoggingConfig,
	scheduler string,
	slotType device.Type,
	slotResourceRequests config.PodSlotResourceRequests,
	resourcePoolConfigs []config.ResourcePoolConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	detMasterIP string,
	detMasterPort int32,
	kubeconfigPath string,
	podStatusUpdateCallback podStatusUpdateCallback,
) *pods {
	loggingTLSConfig := masterTLSConfig
	if loggingConfig.ElasticLoggingConfig != nil {
		loggingTLSConfig = loggingConfig.ElasticLoggingConfig.Security.TLS
	}
	p := &pods{
		wg: waitgroupx.WithContext(context.Background()),

		namespace:                    namespace,
		namespaceToPoolName:          namespaceToPoolName,
		masterServiceName:            masterServiceName,
		masterTLSConfig:              masterTLSConfig,
		scheduler:                    scheduler,
		loggingTLSConfig:             loggingTLSConfig,
		loggingConfig:                loggingConfig,
		podNameToPodHandler:          make(map[string]*pod),
		podNameToResourcePool:        make(map[string]string),
		containerIDToPodName:         make(map[string]string),
		containerIDToSchedulingState: make(map[string]sproto.SchedulingState),
		podNameToContainerID:         make(map[string]string),
		podHandlerToMetadata:         make(map[*pod]podMetadata),
		slotType:                     slotType,
		slotResourceRequests:         slotResourceRequests,
		resourcePoolConfigs:          resourcePoolConfigs,
		baseContainerDefaults:        taskContainerDefaults,
		detMasterIP:                  detMasterIP,
		detMasterPort:                detMasterPort,
		currentNodes:                 make(map[string]*k8sV1.Node),
		nodeToSystemResourceRequests: make(map[string]int64),
		podInterfaces:                make(map[string]typedV1.PodInterface),
		configMapInterfaces:          make(map[string]typedV1.ConfigMapInterface),
		syslog:                       logrus.WithField("namespace", namespace),
		podStatusUpdateCallback:      podStatusUpdateCallback,

		kubeconfigPath: kubeconfigPath,
	}

	if err := p.startClientSet(); err != nil {
		panic(err)
	}
	if err := p.getMasterIPAndPort(); err != nil {
		panic(err)
	}
	if err := p.getSystemResourceRequests(); err != nil {
		panic(err)
	}

	p.startResourceRequestQueue()

	if err := p.deleteDoomedKubernetesResources(); err != nil {
		panic(err)
	}

	err := p.startPodInformer()
	if err != nil {
		panic(err)
	}

	err = p.startNodeInformer()
	switch {
	case err != nil && k8error.IsForbidden(err):
		p.syslog.Warnf("unable to start node informer due to permission error,"+
			"some features will be degraded: %s", err,
		)
	case err != nil:
		panic(err)
	}

	err = p.startEventListeners()
	if err != nil {
		panic(err)
	}

	err = p.startPreemptionListeners()
	if err != nil {
		panic(err)
	}

	return p
}

// StartTaskPod notifies the pods actor to start a pod with the task spec.
type StartTaskPod struct {
	Req          *sproto.AllocateRequest
	AllocationID model.AllocationID
	Spec         tasks.TaskSpec
	Slots        int
	Rank         int
	ResourcePool string
	Namespace    string

	LogContext logger.Context
}

func (p *pods) StartTaskPod(msg StartTaskPod) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.receiveStartTaskPod(msg)
}

func (p *pods) ChangePriority(podID cproto.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.receivePriorityChange(podID)
}

func (p *pods) ChangePosition(podID cproto.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.receivePositionChange(podID)
}

func (p *pods) KillPod(podID cproto.ID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.receiveKillPod(podID)
}

func (p *pods) SummarizeResources(msg SummarizeResources) (*PodsInfo, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.receiveResourceSummarize(msg)
}

func (p *pods) ReattachAllocationPods(msg reattachAllocationPods) ([]reattachPodResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.reattachAllocationPods(msg)
}

func (p *pods) RefreshPodStates(msg refreshPodStates) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refreshPodStates(msg.allocationID)
}

func (p *pods) GetSlots(msg *apiv1.GetSlotsRequest) *apiv1.GetSlotsResponse {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.handleGetSlotsRequest(msg.AgentId)
}

func (p *pods) GetSlot(msg *apiv1.GetSlotRequest) *apiv1.GetSlotResponse {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.handleGetSlotRequest(msg.AgentId, msg.SlotId)
}

func (p *pods) HealthStatus() model.HealthStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, podInterface := range p.podInterfaces {
		_, err := podInterface.List(context.TODO(), metaV1.ListOptions{Limit: 1})
		if err != nil {
			p.syslog.WithError(err).Error("kubernetes resource manager marked as unhealthy")
			return model.Unhealthy
		}
		return model.Healthy
	}

	logrus.Error("expected podInterfaces to be non empty")
	return model.Unhealthy
}

func (p *pods) GetAgents() *apiv1.GetAgentsResponse {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.handleGetAgentsRequest()
}

func (p *pods) GetAgent(msg *apiv1.GetAgentRequest) *apiv1.GetAgentResponse {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.handleGetAgentRequest(msg.AgentId)
}

func (p *pods) EnableAgent(msg *apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.enableNode(msg.AgentId)
}

func (p *pods) DisableAgent(msg *apiv1.DisableAgentRequest) (*apiv1.DisableAgentResponse, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.disableNode(msg.AgentId, msg.Drain)
}

func (p *pods) CreateNamespace(autoCreateNamespace bool, namespaceName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.handleCreateNamespaceRequest(autoCreateNamespace, namespaceName)
}

func readClientConfig(kubeconfigPath string) (*rest.Config, error) {
	if len(kubeconfigPath) == 0 {
		// The default in-cluster case.  Internally, k8s.io/client-go/rest is going to look for
		// environment variables:
		//   - KUBERNETES_SERVICE_HOST
		//   - KUBERNETES_SERVICE_PORT
		// and it expects to find files:
		//   - /var/run/secrets/kubernetes.io/serviceaccount/token
		//   - /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
		return rest.InClusterConfig()
	}

	if parts := strings.Split(kubeconfigPath, string(os.PathSeparator)); parts[0] == "~" {
		parts[0] = homedir.HomeDir()
		expanded := filepath.Join(parts...)
		logrus.Infof("expanding kubeconfig path from %s to %s", kubeconfigPath, expanded)
		kubeconfigPath = expanded
	}

	bs, err := os.ReadFile(kubeconfigPath) // #nosec G304 // User must have fs access to set this config var anyway.
	if err != nil {
		return nil, fmt.Errorf("reading kubeconfig at %s: %w", kubeconfigPath, err)
	}

	cl, err := clientcmd.RESTConfigFromKubeConfig(bs)
	if err != nil {
		return nil, fmt.Errorf("building rest.Config from kubeconfig at %s: %w", kubeconfigPath, err)
	}
	return cl, nil
}

func (p *pods) startClientSet() error {
	config, err := readClientConfig(p.kubeconfigPath)
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

	p.syslog.Infof("kubernetes clientSet initialized")
	return nil
}

func (p *pods) getMasterIPAndPort() error {
	if p.detMasterIP != "" && p.detMasterPort != 0 {
		// Master ip and port were manually configured. For special circumstances, e.g., the master is running
		// outside of this cluster (happens in development or when we spread across multiple k8s clusters).
		return nil
	}
	masterService, err := p.clientSet.CoreV1().Services(p.namespace).Get(
		context.TODO(), p.masterServiceName, metaV1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get master service")
	}

	p.detMasterIP = masterService.Spec.ClusterIP
	p.detMasterPort = masterService.Spec.Ports[0].Port
	p.syslog.Infof("master URL set to %s:%d", p.detMasterIP, p.detMasterPort)
	return nil
}

func (p *pods) getSystemResourceRequests() error {
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

func (p *pods) reattachAllocationPods(msg reattachAllocationPods) ([]reattachPodResponse, error) {
	listOptions := metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", determinedLabel, msg.allocationID),
	}

	pods, err := p.listPodsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "error listing pods checking if they can be restored")
	}

	configMaps, err := p.listConfigMapsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return nil, errors.Wrap(err, "error listing config maps checking if they can be restored")
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
						p.deleteKubernetesResources(pods, configMaps)
						return nil, fmt.Errorf("pod missing config map %s", pod.Name)
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
		p.deleteKubernetesResources(pods, configMaps)
		return nil, fmt.Errorf("not enough pods found for allocation expected %d got %d instead",
			msg.numPods, len(k8sPods))
	}

	if err := p.dontReattachQueuedPreAgentDisabledPods(pods, configMaps); err != nil {
		return nil, err
	}

	var restoreResponses []reattachPodResponse
	for i, containerID := range containerIDs {
		resp, err := p.reattachPod(msg.req, msg.allocationID, resourcePool, containerID,
			k8sPods[i], ports[i], msg.slots, msg.logContext)
		if err != nil {
			p.deleteKubernetesResources(pods, configMaps)
			return nil, errors.Wrapf(err,
				"error restoring pod with containerID %s", containerID)
		}
		restoreResponses = append(restoreResponses, resp)
	}

	return restoreResponses, nil
}

func (p *pods) dontReattachQueuedPreAgentDisabledPods(
	pods *k8sV1.PodList, configMaps *k8sV1.ConfigMapList,
) error {
	// This is needed to label pods created before Determined supported k8s agent enable disable.
	// We will not reattach pods that are queued and don't have the affinity that respects
	// agent disabling. Not many people should be relying on this feature when this will be released
	// since it was behind _agent_reattach_enabled until the version this is also released on.
	// We can't patch the pods with the needed field, as a limitation of Kubernetes.
	for _, pod := range pods.Items {
		pod := pod
		if pod.Spec.NodeName == "" { // Only do this for pods not assigned to a node yet.
			before := pod.DeepCopy()
			addNodeDisabledAffinityToPodSpec(&pod, clusterIDNodeLabel())

			if !reflect.DeepEqual(pod.Spec, before.Spec) {
				p.deleteKubernetesResources(pods, configMaps)
				return fmt.Errorf(
					"unable to restore pod %s since it was queued and does not have the needed "+
						"Determined's affinity to prevent scheduling on disabled nodes. "+
						"This is expected to happen on allocations with queued pods "+
						"when upgrading from before 0.25.1 "+
						"to after or equal to 0.26.1", pod.Name)
			}
		}
	}

	return nil
}

func (p *pods) reattachPod(
	req *sproto.AllocateRequest,
	allocationID model.AllocationID,
	resourcePool string,
	containerID string,
	pod *k8sV1.Pod,
	ports []int,
	slots int,
	logContext logger.Context,
) (reattachPodResponse, error) {
	startMsg := StartTaskPod{
		Req:          req,
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
		p.detMasterIP,
		p.detMasterPort,
		p.masterTLSConfig,
		p.loggingTLSConfig,
		p.loggingConfig,
		p.podInterfaces[pod.Namespace],
		p.configMapInterfaces[pod.Namespace],
		p.resourceRequestQueue,
		p.slotType,
		p.slotResourceRequests,
		p.scheduler,
	)

	newPodHandler.restore = true
	newPodHandler.podName = pod.Name
	newPodHandler.configMapName = pod.Name
	newPodHandler.ports = ports

	state, err := newPodHandler.getPodState(pod, newPodHandler.containerNames)
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

	err = newPodHandler.start()
	if err != nil {
		return reattachPodResponse{}, fmt.Errorf("reattaching pod: %w", err)
	}

	p.podNameToPodHandler[pod.Name] = newPodHandler
	p.podNameToResourcePool[pod.Name] = resourcePool
	p.containerIDToPodName[containerID] = pod.Name
	p.podNameToContainerID[pod.Name] = containerID
	p.containerIDToSchedulingState[containerID] = sproto.SchedulingStateQueued
	p.podHandlerToMetadata[newPodHandler] = podMetadata{
		podName:     pod.Name,
		containerID: containerID,
	}

	return reattachPodResponse{containerID: containerID, started: started}, nil
}

func (p *pods) refreshPodStates(allocationID model.AllocationID) error {
	if allocationID == "" {
		return fmt.Errorf("invalid call: allocationID missing")
	}

	pods, err := p.listPodsInAllNamespaces(context.TODO(), metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", determinedLabel, allocationID),
	})
	if err != nil {
		return errors.Wrap(err, "error listing pods checking if they can be restored")
	}

	for _, pod := range pods.Items {
		if _, ok := p.namespaceToPoolName[pod.Namespace]; !ok {
			continue
		}
		pod := pod
		p.podStatusCallback(watch.Event{Object: &pod})
	}
	return nil
}

func (p *pods) deleteKubernetesResources(
	pods *k8sV1.PodList, configMaps *k8sV1.ConfigMapList,
) {
	for _, pod := range pods.Items {
		p.resourceRequestQueue.deleteKubernetesResources(pod.Namespace, pod.Name, "")
	}

	for _, configMap := range configMaps.Items {
		p.resourceRequestQueue.deleteKubernetesResources(configMap.Namespace, "", configMap.Name)
	}
}

func (p *pods) deleteDoomedKubernetesResources() error {
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
			p.syslog.Debugf("deleting pod '%s' without environment variable '%s'",
				pod.Name, resourcePoolEnvVar)
			toKillPods.Items = append(toKillPods.Items, pod)
			continue
		}

		if !openAllocationIDs.Contains(model.AllocationID(pod.Labels[determinedLabel])) {
			p.syslog.Warnf("deleting pod '%s', did not find open allocation '%s'",
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

		p.syslog.Debugf("Deleting config map '%s' did not find a matching pod that will be restored",
			cm.Name)
		toKillConfigMaps.Items = append(toKillConfigMaps.Items, cm)
	}

	p.deleteKubernetesResources(toKillPods, toKillConfigMaps)
	return nil
}

func (p *pods) startPodInformer() error {
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
				p.podStatusCallback(event)
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

func (p *pods) startEventListeners() error {
	for namespace := range p.namespaceToPoolName {
		l, err := newEventInformer(
			context.TODO(),
			p.clientSet.CoreV1().Events(namespace),
			namespace,
			func(event watch.Event) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.eventStatusCallback(event)
			})
		if err != nil {
			return err
		}
		go l.run(context.TODO())
	}
	return nil
}

func (p *pods) startPreemptionListeners() error {
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
				p.preemptionCallback(event)
			})
		if err != nil {
			return err
		}
		go l.run(context.TODO())
	}
	return nil
}

func (p *pods) startResourceRequestQueue() {
	failures := make(chan resourcesRequestFailure, 16)
	p.resourceRequestQueue = startRequestQueue(p.podInterfaces, p.configMapInterfaces, failures)
	p.wg.Go(func(ctx context.Context) {
		for {
			select {
			case failure := <-failures:
				p.handleResourceRequestFailure(failure)
			case <-ctx.Done():
				return
			}
		}
	})
}

func (p *pods) handleResourceRequestFailure(msg resourcesRequestFailure) {
	p.mu.Lock()
	defer p.mu.Unlock()

	podName := msg.getPodName()
	podHandler, ok := p.podNameToPodHandler[podName]
	if !ok {
		p.syslog.Warnf("received resource request error for unregistered pod %s", podName)
		return
	}

	switch msg := msg.(type) {
	case resourceCreationFailed:
		podHandler.receiveResourceCreationFailed(msg)
	case resourceCreationCancelled:
		podHandler.receiveResourceCreationCancelled()
	case resourceDeletionFailed:
		podHandler.receiveResourceDeletionFailed(msg)
	default:
		panic(fmt.Sprintf("unexpected message %T", msg))
	}

	err := p.cleanUpPodHandler(podHandler)
	if err != nil {
		p.syslog.WithError(err).Error("cleaning up pod handler after resource request failure")
	}
}

func (p *pods) receiveStartTaskPod(msg StartTaskPod) error {
	newPodHandler := newPod(
		msg,
		msg.Spec.ClusterID,
		p.clientSet,
		msg.Namespace,
		p.detMasterIP,
		p.detMasterPort,
		p.masterTLSConfig,
		p.loggingTLSConfig,
		p.loggingConfig,
		p.podInterfaces[msg.Namespace],
		p.configMapInterfaces[msg.Namespace],
		p.resourceRequestQueue,
		p.slotType,
		p.slotResourceRequests,
		p.scheduler,
	)

	if _, alreadyExists := p.podNameToPodHandler[newPodHandler.podName]; alreadyExists {
		return errors.Errorf(
			"attempting to register same pod name: %s multiple times", newPodHandler.podName)
	}

	err := newPodHandler.start()
	if err != nil {
		return fmt.Errorf("creating pod: %w", err)
	}

	p.podNameToPodHandler[newPodHandler.podName] = newPodHandler
	p.podNameToResourcePool[newPodHandler.podName] = msg.ResourcePool
	p.containerIDToPodName[msg.Spec.ContainerID] = newPodHandler.podName
	p.podNameToContainerID[newPodHandler.podName] = msg.Spec.ContainerID
	p.containerIDToSchedulingState[msg.Spec.ContainerID] = sproto.SchedulingStateQueued
	p.podHandlerToMetadata[newPodHandler] = podMetadata{
		podName:     newPodHandler.podName,
		containerID: msg.Spec.ContainerID,
	}

	return nil
}

func (p *pods) podStatusCallback(event watch.Event) {
	pod, ok := event.Object.(*k8sV1.Pod)
	if !ok {
		p.syslog.Warnf("error converting event of type %T to *k8sV1.Pod: %+v", event, event)
		return
	}
	syslog := p.syslog.WithField("pod", pod.Name)
	syslog.WithField("event.Type", event.Type).Debug("received pod informer event")

	podHandler, ok := p.podNameToPodHandler[pod.Name]
	if !ok {
		syslog.Debug("received status update for un-registered pod")
		return
	}

	state, err := podHandler.podStatusUpdate(pod)
	switch {
	case err != nil:
		syslog.WithError(err).Error("error processing pod status update")
		err := p.cleanUpPodHandler(podHandler)
		if err != nil {
			syslog.WithError(err).Error("unable to cleanup pod handler after update error")
		}
		return
	case state == cproto.Terminated:
		err := p.cleanUpPodHandler(podHandler)
		if err != nil {
			syslog.WithError(err).Error("unable to cleanup pod handler after termination")
		}
	}

	if containerID, ok := p.podNameToContainerID[pod.Name]; ok {
		if state, ok := p.containerIDToSchedulingState[containerID]; ok {
			currState := sproto.SchedulingStateQueued
			if pod.Status.Phase == "Running" {
				currState = sproto.SchedulingStateScheduled
			}
			if currState != state {
				p.containerIDToSchedulingState[containerID] = currState
				go p.podStatusUpdateCallback(sproto.UpdatePodStatus{
					ContainerID: containerID,
					State:       currState,
				})
			}
		}
	}
}

var (
	clusterID string
	once      sync.Once
)

func setClusterID(s string) {
	once.Do(func() {
		clusterID = s
	})
}

func clusterIDNodeLabel() string {
	return fmt.Sprintf("determined.ai/cluster-id-%s", clusterID)
}

const (
	noExecuteNodeLabelValue  = "no-execute"
	noScheduleNodeLabelValue = "no-schedule"
)

func (p *pods) enableNode(
	nodeName string,
) (*apiv1.EnableAgentResponse, error) {
	patch := []byte(fmt.Sprintf(`{
		"metadata": {
			"labels": {
				"%s": null
			}
		}
	}`, clusterIDNodeLabel()))

	_, err := p.clientSet.CoreV1().Nodes().
		Patch(context.TODO(), nodeName, types.StrategicMergePatchType, patch, metaV1.PatchOptions{})
	if k8error.IsForbidden(err) {
		return nil, fmt.Errorf("the Determined master Kubernetes service account " +
			"is missing permissions to patch nodes. " +
			"Enabling or disabling nodes requires this permission, " +
			"however Determined will otherwise still function correctly without " +
			"these Kubernetes permissions")
	} else if err != nil {
		return nil, fmt.Errorf(
			"enabling node %s by removing the Determined no schedule label: %w", nodeName, err)
	}
	p.syslog.Infof("node %s enabled by an user", nodeName)

	n, ok := p.summarizeClusterByNodes()[nodeName]
	if !ok {
		return nil, fmt.Errorf("node %s enabled without error, error getting node summary", nodeName)
	}
	n.Enabled = true
	n.Draining = false
	for slotKey := range n.Slots {
		s := n.Slots[slotKey]
		s.Enabled = n.Enabled
		s.Draining = n.Draining
		n.Slots[slotKey] = s
	}

	return &apiv1.EnableAgentResponse{
		Agent: n.ToProto(),
	}, nil
}

func (p *pods) disableNode(
	nodeName string, shouldDrain bool,
) (*apiv1.DisableAgentResponse, error) {
	labelValue := noExecuteNodeLabelValue
	if shouldDrain {
		labelValue = noScheduleNodeLabelValue
	}

	patchStruct := metaV1.ObjectMeta{
		Labels: map[string]string{clusterIDNodeLabel(): labelValue},
	}
	patch, err := json.Marshal(map[string]any{"metadata": patchStruct})
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON patch %v: %s", patchStruct, err)
	}

	_, err = p.clientSet.CoreV1().Nodes().
		Patch(context.TODO(), nodeName, types.StrategicMergePatchType, patch, metaV1.PatchOptions{})
	if k8error.IsForbidden(err) {
		return nil, fmt.Errorf("the Determined master Kubernetes service account " +
			"is missing permissions to patch nodes. " +
			"Enabling or disabling nodes requires this permission, " +
			"however Determined will otherwise still function correctly without " +
			"these Kubernetes permissions")
	} else if err != nil {
		return nil, fmt.Errorf(
			"disabling node %s by adding the Determined no schedule label: %w", nodeName, err)
	}
	p.syslog.Infof("node %s disabled by an user", nodeName)

	if !shouldDrain { // See note in spec.go about how we could remove killing all pods here.
		if err := p.releaseAllocationsOnDisabledNode(nodeName); err != nil {
			return nil, fmt.Errorf(
				"node disabled without error, error killing existing pod on node: %w", err)
		}
	}

	n, ok := p.summarizeClusterByNodes()[nodeName]
	if !ok {
		return nil, fmt.Errorf("node %s disabled without error, error getting node summary", nodeName)
	}
	n.Enabled = false
	n.Draining = shouldDrain
	for slotKey := range n.Slots {
		s := n.Slots[slotKey]
		s.Enabled = n.Enabled
		s.Draining = n.Draining
		n.Slots[slotKey] = s
	}

	return &apiv1.DisableAgentResponse{
		Agent: n.ToProto(),
	}, nil
}

func (p *pods) releaseAllocationsOnDisabledNode(nodeName string) error {
	listOptions := metaV1.ListOptions{
		LabelSelector: determinedLabel,
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	}
	pods, err := p.listPodsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return fmt.Errorf("listing pods on node %s: %w", nodeName, err)
	}

	notifiedAllocations := make(map[model.AllocationID]bool)
	for _, pod := range pods.Items {
		podHandler, ok := p.podNameToPodHandler[pod.Name]
		if !ok {
			p.syslog.Warnf(
				"during node disable couldn't find pod %s's actor to kill", pod.Name)
			continue
		}

		p.syslog.Infof(
			"stopping pod %s because node %s was disabled without drain option", pod.Name, nodeName)
		if notifiedAllocations[podHandler.allocationID] {
			continue
		}

		rmevents.Publish(podHandler.allocationID, &sproto.ReleaseResources{
			Reason:    "node disabled without drain",
			ForceKill: true,
		})
		notifiedAllocations[podHandler.allocationID] = true
	}

	return nil
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

func (p *pods) eventStatusCallback(event watch.Event) {
	newEvent, ok := event.Object.(*k8sV1.Event)
	if !ok {
		p.syslog.Warnf("error converting object type %T to *k8sV1.Event: %+v", event, event)
		return
	}

	syslog := p.syslog.WithFields(logrus.Fields{
		"name": newEvent.InvolvedObject.Name,
		"kind": newEvent.InvolvedObject.Kind,
	})

	syslog.Debugf("listener got new event: %s", newEvent.Message)
	ref, ok := p.podNameToPodHandler[newEvent.InvolvedObject.Name]
	if !ok {
		// We log at the debug level because we are unable to filter
		// pods based on their labels the way we do with pod status updates.
		syslog.Debug("received pod event for an un-registered pod")
		return
	}

	ref.podEventUpdate(newEvent)
}

func (p *pods) receiveResourceSummarize(msg SummarizeResources) (*PodsInfo, error) {
	summary, err := p.summarize()
	if err != nil {
		return nil, err
	}

	slots := 0
	if len(msg.PoolName) > 0 {
		slots = numSlots(summary[msg.PoolName].Slots)
	} else {
		for _, pool := range summary {
			slots += numSlots(pool.Slots)
		}
	}
	return &PodsInfo{NumAgents: len(summary), SlotsAvailable: slots}, nil
}

func (p *pods) preemptionCallback(event watch.Event) {
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
	ref.PreemptTaskPod()
}

func (p *pods) verifyPodAndGetRef(podID string) *pod {
	podName, ok := p.containerIDToPodName[podID]
	if !ok {
		p.syslog.WithField("pod-id", podID).Debug(
			"received change priority command for unregistered container id")
		return nil
	}
	ref, ok := p.podNameToPodHandler[podName]
	if !ok {
		p.syslog.WithField("pod-id", podID).Debug(
			"received change priority command for unregistered container id")
		return nil
	}

	return ref
}

func (p *pods) receivePriorityChange(podID cproto.ID) {
	ref := p.verifyPodAndGetRef(podID.String())
	if ref != nil {
		ref.ChangePriority()
	}
}

func (p *pods) receivePositionChange(podID cproto.ID) {
	ref := p.verifyPodAndGetRef(podID.String())
	if ref != nil {
		ref.ChangePosition()
	}
}

func (p *pods) receiveKillPod(podID cproto.ID) {
	name, ok := p.containerIDToPodName[podID.String()]
	if !ok {
		// For multi-pod tasks, when the chief pod exits, the scheduler
		// will request to terminate pods all other pods that have
		// notified the scheduler that they have exited.
		p.syslog.WithField("pod-id", podID).Info(
			"received stop pod command for unregistered container id")
		return
	}

	ref, ok := p.podNameToPodHandler[name]
	if !ok {
		p.syslog.WithField("pod-id", podID).Info(
			"received stop pod command for unregistered container id")
		return
	}

	ref.KillTaskPod()
}

func (p *pods) cleanUpPodHandler(podHandler *pod) error {
	podHandler.finalize()

	podInfo, ok := p.podHandlerToMetadata[podHandler]
	if !ok {
		return errors.Errorf("unknown pod handler being deleted %s", podHandler.podName)
	}

	p.syslog.WithField("pod", podInfo.podName).WithField(
		"handler", podHandler.podName).Infof("de-registering pod handler")
	delete(p.podNameToPodHandler, podInfo.podName)
	delete(p.podNameToResourcePool, podInfo.podName)
	delete(p.podNameToContainerID, podInfo.podName)
	delete(p.containerIDToPodName, podInfo.containerID)
	delete(p.containerIDToSchedulingState, podInfo.containerID)
	delete(p.podHandlerToMetadata, podHandler)

	// launch this work async, since we hold the lock and it does API calls.
	p.wg.Go(func(ctx context.Context) {
		name := fmt.Sprintf("%s-priorityclass", podInfo.containerID)
		err := p.clientSet.
			SchedulingV1().
			PriorityClasses().
			Delete(ctx, name, metaV1.DeleteOptions{})
		if err != nil && !k8error.IsNotFound(err) {
			p.syslog.Warnf("Deletion of PriorityClass %s failed.", name)
		}
	})

	return nil
}

func (p *pods) handleGetSlotsRequest(agentID string) *apiv1.GetSlotsResponse {
	agentResp := p.handleGetAgentRequest(agentID)
	if agentResp == nil {
		p.syslog.Warnf("no agent with id %s", agentID)
		return nil
	}
	return &apiv1.GetSlotsResponse{Slots: maps.Values(agentResp.Agent.Slots)}
}

func (p *pods) handleGetSlotRequest(agentID string, slotID string) *apiv1.GetSlotResponse {
	agentResp := p.handleGetAgentRequest(agentID)
	if agentResp == nil {
		p.syslog.Warnf("no agent with id %s", agentID)
		return nil
	}
	slots := agentResp.Agent.Slots
	slot, ok := slots[slotID]
	if !ok {
		// Try converting an index input to a slot and see if that exists (1 to 001).
		tryIndex, err := strconv.Atoi(slotID)
		if s, ok := slots[model.SortableSlotIndex(tryIndex)]; err == nil && ok {
			slot = s
		} else {
			p.syslog.Warnf("no slot with id %s", slotID)
			return nil
		}
	}
	return &apiv1.GetSlotResponse{Slot: slot}
}

func (p *pods) handleGetAgentsRequest() *apiv1.GetAgentsResponse {
	p.getAgentsCacheLock.Lock()
	defer p.getAgentsCacheLock.Unlock()

	if time.Since(p.getAgentsCacheTime) > getAgentsCacheDuration {
		p.getAgentsCacheTime = time.Now()

		nodeSummaries := p.summarizeClusterByNodes()
		_, nodesToPools := p.getNodeResourcePoolMapping(nodeSummaries)

		p.getAgentsCache = &apiv1.GetAgentsResponse{}
		for _, summary := range nodeSummaries {
			summary.ResourcePool = nodesToPools[summary.ID]
			p.getAgentsCache.Agents = append(p.getAgentsCache.Agents, summary.ToProto())
		}
	}

	return p.getAgentsCache
}

func (p *pods) handleGetAgentRequest(agentID string) *apiv1.GetAgentResponse {
	nodeSummaries := p.summarizeClusterByNodes()
	_, nodesToPools := p.getNodeResourcePoolMapping(nodeSummaries)
	agentSummary, ok := nodeSummaries[agentID]
	if !ok {
		// TODO(DET-10029): We should return an error indicating the invalid ID request (rather
		//	than a warn).
		p.syslog.Warnf("no agent with id %s", agentID)
		return nil
	}
	agentSummary.ResourcePool = nodesToPools[agentSummary.ID]
	return &apiv1.GetAgentResponse{Agent: agentSummary.ToProto()}
}

func (p *pods) handleCreateNamespaceRequest(autoCreateNamespace bool, namespaceName string) error {
	var k8sDeterminedLabel = map[string]string{determinedLabel: namespaceName}

	if autoCreateNamespace {
		// If the namespace exists, but has a determined label, keep it. Error out if it doesn't have the label
		namespaceToCreate := k8sV1.Namespace{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name:   namespaceName,
				Labels: k8sDeterminedLabel,
			},
		}

		_, err := p.clientSet.CoreV1().Namespaces().Create(context.Background(), &namespaceToCreate,
			metaV1.CreateOptions{},
		)
		if err != nil {
			if !strings.Contains(err.Error(), "already exists") {
				return fmt.Errorf("error creating namespace %s: %w", namespaceName, err)
			}
		}

	} else {
		// If the namespace doesn't exist, return an error.
		// Remember that quota should not be specified here (which we verify in workspace.py)
		_, err := p.clientSet.CoreV1().Namespaces().Get(context.Background(), namespaceName,
			metaV1.GetOptions{
				TypeMeta: metaV1.TypeMeta{
					Kind:       "Namespace",
					APIVersion: "v1",
				},
			})
		if err != nil {
			return errors.Wrapf(err, "error finding namespace %s", namespaceName)
		}
	}
	return nil
}

// summarize describes pods' available resources. When there's exactly one resource pool, it uses
// the whole cluster's info. Otherwise, it matches nodes to resource pools using taints and
// tolerations to derive that info. This may be cached, so don't use this for decisions
// that require up-to-date information.
func (p *pods) summarize() (map[string]model.AgentSummary, error) {
	p.summarizeCacheLock.Lock()
	defer p.summarizeCacheLock.Unlock()

	if time.Since(p.summarizeCacheTime) > summarizeCacheDuration {
		summary, err := p.computeSummary()
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

			// add default toleration so that autoscaling nodes will still be counted.
			poolTolerations = append(poolTolerations, k8sV1.Toleration{
				Key:               "DeletionCandidateOfClusterAutoscaler",
				Operator:          "Exists",
				Effect:            "PreferNoSchedule",
				TolerationSeconds: nil,
			})
			// If all of a node's taints are tolerated by a pool, that node belongs to the pool.
			if allTaintsTolerated(node.Spec.Taints, poolTolerations) {
				poolsToNodes[poolName] = append(poolsToNodes[poolName], node)
				nodesToPools[node.Name] = append(nodesToPools[node.Name], poolName)
			}
		}
	}

	return poolsToNodes, nodesToPools
}

var programStartTime = time.Now()

func (p *pods) computeSummary() (map[string]model.AgentSummary, error) {
	nodeSummaries := p.summarizeClusterByNodes()

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
			RegisteredTime: programStartTime,
			NumContainers:  numContainersInPool,
			ResourcePool:   []string{poolName},
			Slots:          slots,
		}
	}

	return summaries, nil
}

func (p *pods) summarizeClusterByNodes() map[string]model.AgentSummary {
	var allPods []podNodeInfo

	for _, p := range p.podNameToPodHandler {
		allPods = append(allPods, p.getPodNodeInfo())
	}

	// Separate pods by nodes.
	podByNode := make(map[string][]podNodeInfo, len(allPods))
	for _, podInfo := range allPods {
		if len(podInfo.nodeName) == 0 {
			// If a pod doesn't have a nodeName it means it has not yet
			// been allocated to a node.
			continue
		}
		podByNode[podInfo.nodeName] = append(podByNode[podInfo.nodeName], podInfo)
	}

	nodeToTasks, taskSlots := p.getNonDetSlots(p.slotType)
	summary := make(map[string]model.AgentSummary, len(p.currentNodes))
	for _, node := range p.currentNodes {
		disabledLabel, isDisabled := node.Labels[clusterIDNodeLabel()]
		isDraining := isDisabled && disabledLabel == noScheduleNodeLabelValue

		var numSlots int64
		var deviceType device.Type

		// TODO(DET-10010): slot type per node probably shouldn't be decided from pods literal
		// (which has the same value for all nodes).
		switch p.slotType {
		case device.CPU:
			resources := node.Status.Allocatable[k8sV1.ResourceCPU]
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
					p.syslog.Warnf("too many pods mapping to node %s", node.Name)
					continue
				}

				slotsSummary[model.SortableSlotIndex(curSlot)] = model.SlotSummary{
					ID:        model.SortableSlotIndex(curSlot),
					Device:    device.Device{Type: deviceType},
					Draining:  isDraining,
					Enabled:   !isDisabled,
					Container: podInfo.container,
				}
				curSlot++
			}
		}

		for _, taskName := range nodeToTasks[node.Name] {
			for i := int64(0); i < taskSlots[taskName]; i++ {
				if curSlot >= int(numSlots) {
					p.syslog.Warnf("too many pods mapping to node %s", node.Name)
					continue
				}

				slotsSummary[model.SortableSlotIndex(curSlot)] = model.SlotSummary{
					ID:       model.SortableSlotIndex(curSlot),
					Device:   device.Device{Type: deviceType},
					Draining: isDraining,
					Enabled:  !isDisabled,
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
			slotsSummary[model.SortableSlotIndex(i)] = model.SlotSummary{
				ID:       model.SortableSlotIndex(i),
				Device:   device.Device{Type: deviceType},
				Draining: isDraining,
				Enabled:  !isDisabled,
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
			Draining:       isDraining,
			Enabled:        !isDisabled,
		}
	}

	return summary
}

func (p *pods) getNonDetPods() ([]k8sV1.Pod, error) {
	// TODO(RM-235) use a filter in metaV1.ListOptions. This change gets a lot easier after
	// we have K8s integration tests. Using a filter means we should really talk to a real
	// k8s server. Doing an e2e test for this is possible but would take a lot more work.
	allPods, err := p.listPodsInAllNamespaces(context.TODO(), metaV1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var nonDetPods []k8sV1.Pod
	for _, p := range allPods.Items {
		_, isDet := p.Labels[determinedLabel]
		_, isDetSystem := p.Labels[determinedSystemLabel]

		if !(isDet || isDetSystem) {
			if p.Spec.NodeName != "" {
				nonDetPods = append(nonDetPods, p)
			}
		}
	}
	return nonDetPods, nil
}

func (p *pods) getNonDetSlots(deviceType device.Type) (map[string][]string, map[string]int64) {
	nodeToTasks := make(map[string][]string, len(p.currentNodes))
	taskSlots := make(map[string]int64)

	nonDetPods, err := p.getNonDetPods()
	if err != nil {
		p.syslog.WithError(err).Warn("getting non determined pods, " +
			"this may cause slots to look free when they are in use")
	}

	if len(nonDetPods) == 0 {
		return nodeToTasks, taskSlots
	}
	for _, node := range p.currentNodes {
		nodeToTasks[node.Name] = []string{}
	}

	// Ignore pods not yet scheduled on a node.
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
