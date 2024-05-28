package kubernetesrm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	batchV1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/informers"
	typedBatchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"k8s.io/client-go/tools/cache"

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

type jobMetadata struct {
	jobName      string
	allocationID model.AllocationID
}

type jobSchedulingStateCallbackFn func(jobSchedulingStateChanged)

type jobSchedulingStateChanged struct {
	AllocationID model.AllocationID
	NumPods      int
	State        sproto.SchedulingState
}

// High lever overview of the actors within the kubernetes package:
//
//	jobsService
//	  +- pod(s): manages pod lifecycle. One per container in a task.
//	     +- podLogStreamer: stream logs for a specific pod.
//	  +- informer: sends updates about pod states
//	  +- events: sends updates about kubernetes events.
//	  +- requestQueue: queues requests to create / delete kubernetes resources.
//	     +- requestProcessingWorkers: processes request to create / delete kubernetes resources.
//
// TODO(DET-10011): Give this literal a more intuitive name.
type jobsService struct {
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

	clientSet       k8sClient.Interface
	detMasterIP     string
	detMasterPort   int32
	masterTLSConfig model.TLSClientConfig

	resourceRequestQueue              *requestQueue
	jobNameToJobHandler               map[string]*job
	jobNameToResourcePool             map[string]string
	jobNameToPodNameToSchedulingState map[string]map[string]sproto.SchedulingState
	allocationIDToJobName             map[model.AllocationID]string
	jobHandlerToMetadata              map[*job]jobMetadata
	nodeToSystemResourceRequests      map[string]int64

	currentNodes map[string]*k8sV1.Node

	podInterfaces       map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface
	jobInterfaces       map[string]typedBatchV1.JobInterface

	// TODO(RM-236) make one cache and make this code more straightforward.
	summarizeCacheLock sync.RWMutex
	summarizeCache     summarizeResult
	summarizeCacheTime time.Time
	getAgentsCacheLock sync.Mutex
	getAgentsCache     *apiv1.GetAgentsResponse
	getAgentsCacheTime time.Time

	syslog *logrus.Entry

	jobSchedulingStateCallback jobSchedulingStateCallbackFn
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

type reattachJobRequest struct {
	req          *sproto.AllocateRequest
	numPods      int
	allocationID model.AllocationID
	slots        int
	logContext   logger.Context
}

type reattachJobResponse struct {
	started *sproto.ResourcesStarted
}

// newJobsService creates a new pod service for launching, querying and interacting with k8s pods.
func newJobsService(
	namespace string,
	namespaceToPoolName map[string]string,
	masterServiceName string,
	masterTLSConfig model.TLSClientConfig,
	scheduler string,
	slotType device.Type,
	slotResourceRequests config.PodSlotResourceRequests,
	resourcePoolConfigs []config.ResourcePoolConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	detMasterIP string,
	detMasterPort int32,
	kubeconfigPath string,
	jobSchedulingStateCallback jobSchedulingStateCallbackFn,
) *jobsService {
	p := &jobsService{
		wg: waitgroupx.WithContext(context.Background()),

		namespace:                         namespace,
		namespaceToPoolName:               namespaceToPoolName,
		masterServiceName:                 masterServiceName,
		masterTLSConfig:                   masterTLSConfig,
		scheduler:                         scheduler,
		jobNameToJobHandler:               make(map[string]*job),
		jobNameToResourcePool:             make(map[string]string),
		allocationIDToJobName:             make(map[model.AllocationID]string),
		jobNameToPodNameToSchedulingState: make(map[string]map[string]sproto.SchedulingState),
		jobHandlerToMetadata:              make(map[*job]jobMetadata),
		slotType:                          slotType,
		slotResourceRequests:              slotResourceRequests,
		resourcePoolConfigs:               resourcePoolConfigs,
		baseContainerDefaults:             taskContainerDefaults,
		detMasterIP:                       detMasterIP,
		detMasterPort:                     detMasterPort,
		currentNodes:                      make(map[string]*k8sV1.Node),
		nodeToSystemResourceRequests:      make(map[string]int64),
		podInterfaces:                     make(map[string]typedV1.PodInterface),
		configMapInterfaces:               make(map[string]typedV1.ConfigMapInterface),
		jobInterfaces:                     make(map[string]typedBatchV1.JobInterface),
		syslog:                            logrus.WithField("namespace", namespace),
		jobSchedulingStateCallback:        jobSchedulingStateCallback,

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

	err := p.startNodeInformer()
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

	var cacheSyncs []cache.InformerSynced
	for namespace := range p.namespaceToPoolName {
		factory := informers.NewSharedInformerFactoryWithOptions(p.clientSet, time.Hour, informers.WithNamespace(namespace))

		jobsInformer := factory.Batch().V1().Jobs()
		jobsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.jobUpdatedCallback(obj)
			},
			UpdateFunc: func(_, obj interface{}) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.jobUpdatedCallback(obj)
			},

			// If a job is deleted out from under us, this is the only hook we have to not
			// leave our workloads running or pending forever.
			DeleteFunc: func(obj interface{}) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.jobDeletedCallback(obj)
			},
		})
		cacheSyncs = append(cacheSyncs, jobsInformer.Informer().HasSynced)

		podsInformer := factory.Core().V1().Pods()
		podsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.podStatusCallback(obj)
			},
			UpdateFunc: func(_, obj interface{}) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.podStatusCallback(obj)
			},

			// If a pod is deleted out from under us, it is nice to let the user know that
			// is what happened.
			DeleteFunc: func(obj interface{}) {
				p.mu.Lock()
				defer p.mu.Unlock()
				p.podDeletedCallback(obj)
			},
		})
		cacheSyncs = append(cacheSyncs, podsInformer.Informer().HasSynced)

		factory.Start(nil)
	}
	if !cache.WaitForCacheSync(nil, cacheSyncs...) {
		panic("failed to wait for cache sync for jobs informer")
	}

	return p
}

// StartJob notifies the pods actor to start a pod with the task spec.
type StartJob struct {
	Req          *sproto.AllocateRequest
	AllocationID model.AllocationID
	Spec         tasks.TaskSpec
	Slots        int
	Rank         int
	ResourcePool string
	Namespace    string

	NumPods int

	LogContext logger.Context
}

func (j *jobsService) StartJob(msg StartJob) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.startJob(msg)
}

func (j *jobsService) ChangePriority(id model.AllocationID) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.receivePriorityChange(id)
}

func (j *jobsService) ChangePosition(id model.AllocationID) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.receivePositionChange(id)
}

func (j *jobsService) KillJob(id model.AllocationID) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.receiveKill(id)
}

func (j *jobsService) SummarizeResources(msg SummarizeResources) (*PodsInfo, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.receiveResourceSummarize(msg)
}

func (j *jobsService) ReattachJob(msg reattachJobRequest) (reattachJobResponse, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.reattachJob(msg)
}

func (j *jobsService) RefreshStates(allocationID model.AllocationID) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	err := j.refreshJobState(allocationID)
	if err != nil {
		return err
	}
	return j.refreshPodStates(allocationID)
}

func (j *jobsService) GetSlots(msg *apiv1.GetSlotsRequest) *apiv1.GetSlotsResponse {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.handleGetSlotsRequest(msg.AgentId)
}

func (j *jobsService) GetSlot(msg *apiv1.GetSlotRequest) *apiv1.GetSlotResponse {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.handleGetSlotRequest(msg.AgentId, msg.SlotId)
}

func (j *jobsService) HealthStatus() model.HealthStatus {
	j.mu.Lock()
	defer j.mu.Unlock()
	for _, podInterface := range j.podInterfaces {
		_, err := podInterface.List(context.TODO(), metaV1.ListOptions{Limit: 1})
		if err != nil {
			j.syslog.WithError(err).Error("kubernetes resource manager marked as unhealthy")
			return model.Unhealthy
		}
		return model.Healthy
	}

	logrus.Error("expected jobInterface to be non empty")
	return model.Unhealthy
}

func (j *jobsService) GetAgents() *apiv1.GetAgentsResponse {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.handleGetAgentsRequest()
}

func (j *jobsService) GetAgent(msg *apiv1.GetAgentRequest) *apiv1.GetAgentResponse {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.handleGetAgentRequest(msg.AgentId)
}

func (j *jobsService) EnableAgent(msg *apiv1.EnableAgentRequest) (*apiv1.EnableAgentResponse, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.enableNode(msg.AgentId)
}

func (j *jobsService) DisableAgent(msg *apiv1.DisableAgentRequest) (*apiv1.DisableAgentResponse, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.disableNode(msg.AgentId, msg.Drain)
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

func (j *jobsService) startClientSet() error {
	config, err := readClientConfig(j.kubeconfigPath)
	if err != nil {
		return errors.Wrap(err, "error building kubernetes config")
	}

	j.clientSet, err = k8sClient.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to initialize kubernetes clientSet")
	}

	for _, ns := range append(maps.Keys(j.namespaceToPoolName), j.namespace) {
		j.podInterfaces[ns] = j.clientSet.CoreV1().Pods(ns)
		j.configMapInterfaces[ns] = j.clientSet.CoreV1().ConfigMaps(ns)
		j.jobInterfaces[ns] = j.clientSet.BatchV1().Jobs(ns)
	}

	j.syslog.Infof("kubernetes clientSet initialized")
	return nil
}

func (j *jobsService) getMasterIPAndPort() error {
	if j.detMasterIP != "" && j.detMasterPort != 0 {
		// Master ip and port were manually configured. For special circumstances, e.g., the master is running
		// outside of this cluster (happens in development or when we spread across multiple k8s clusters).
		return nil
	}
	masterService, err := j.clientSet.CoreV1().Services(j.namespace).Get(
		context.TODO(), j.masterServiceName, metaV1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get master service")
	}

	j.detMasterIP = masterService.Spec.ClusterIP
	j.detMasterPort = masterService.Spec.Ports[0].Port
	j.syslog.Infof("master URL set to %s:%d", j.detMasterIP, j.detMasterPort)
	return nil
}

func (j *jobsService) getSystemResourceRequests() error {
	systemPods, err := j.podInterfaces[j.namespace].List(
		context.TODO(), metaV1.ListOptions{LabelSelector: determinedSystemLabel})
	if err != nil {
		return errors.Wrap(err, "failed to get system pods")
	}

	for _, systemPod := range systemPods.Items {
		for _, container := range systemPod.Spec.Containers {
			j.nodeToSystemResourceRequests[systemPod.Spec.NodeName] += container.Resources.Requests.Cpu().
				MilliValue()
		}
	}
	return nil
}

func (j *jobsService) reattachJob(msg reattachJobRequest) (reattachJobResponse, error) {
	listOptions := metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", determinedLabel, msg.allocationID),
	}

	jobs, err := j.listJobsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return reattachJobResponse{}, errors.Wrap(err, "error listing pods checking if they can be restored")
	}

	configMaps, err := j.listConfigMapsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return reattachJobResponse{}, errors.Wrap(err, "error listing config maps checking if they can be restored")
	}
	existingConfigMaps := make(set.Set[string])
	for _, cm := range configMaps.Items {
		if _, ok := j.namespaceToPoolName[cm.Namespace]; !ok {
			continue
		}
		existingConfigMaps.Insert(cm.Name)
	}

	if len(jobs.Items) == 0 {
		return reattachJobResponse{}, fmt.Errorf("did not find job for allocation %s", msg.allocationID)
	} else if len(jobs.Items) > 1 {
		return reattachJobResponse{}, fmt.Errorf("found multiple allocation jobs for allocation %s", msg.allocationID)
	}
	job := jobs.Items[0]

	resourcePool, ok := job.Labels[resourcePoolLabel]
	if !ok {
		return reattachJobResponse{}, fmt.Errorf("could not recover resource pool for %s", msg.allocationID)
	}

	resp, err := j.recreateJobHandler(
		msg.req,
		msg.allocationID,
		resourcePool,
		&job,
		msg.slots,
		msg.numPods,
		msg.logContext,
	)
	if err != nil {
		j.deleteKubernetesResources(jobs, configMaps)
		return reattachJobResponse{}, errors.Wrapf(err, "error restoring pod with allocation ID %s", msg.allocationID)
	}
	return resp, nil
}

func (j *jobsService) recreateJobHandler(
	req *sproto.AllocateRequest,
	allocationID model.AllocationID,
	resourcePool string,
	job *batchV1.Job,
	slots int,
	numPods int,
	logContext logger.Context,
) (reattachJobResponse, error) {
	startMsg := StartJob{
		Req:          req,
		AllocationID: allocationID,
		Spec: tasks.TaskSpec{
			// This gets used in reattach to find the job by label its determinedLabel.
			AllocationID: string(allocationID),
			ContainerID:  req.AllocationID.String(), // ContainerID is non-sense, make a better abstraction.
		},
		Slots:        slots,
		NumPods:      numPods,
		ResourcePool: resourcePool,
		LogContext:   logContext,
	}

	newJobHandler := newJob(
		startMsg,
		startMsg.Spec.ClusterID,
		j.clientSet,
		job.Namespace,
		j.detMasterIP,
		j.detMasterPort,
		j.masterTLSConfig,
		j.podInterfaces[job.Namespace],
		j.configMapInterfaces[job.Namespace],
		j.resourceRequestQueue,
		j.slotType,
		j.slotResourceRequests,
		j.scheduler,
	)

	newJobHandler.restore = true
	newJobHandler.jobName = job.Name
	newJobHandler.configMapName = job.Name

	err := newJobHandler.start()
	if err != nil {
		return reattachJobResponse{}, fmt.Errorf("reattaching pod: %w", err)
	}

	j.jobNameToJobHandler[job.Name] = newJobHandler
	j.jobNameToResourcePool[job.Name] = resourcePool
	j.allocationIDToJobName[newJobHandler.req.AllocationID] = job.Name
	j.jobNameToPodNameToSchedulingState[job.Name] = make(map[string]sproto.SchedulingState)
	j.jobHandlerToMetadata[newJobHandler] = jobMetadata{
		jobName:      job.Name,
		allocationID: newJobHandler.req.AllocationID,
	}

	return reattachJobResponse{started: nil}, nil
}

func (j *jobsService) refreshJobState(allocationID model.AllocationID) error {
	if allocationID == "" {
		return fmt.Errorf("invalid call: allocationID missing")
	}

	jobs, err := j.listJobsInAllNamespaces(context.TODO(), metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", determinedLabel, allocationID),
	})
	if err != nil {
		return errors.Wrap(err, "error listing pods checking if they can be restored")
	}

	for _, job := range jobs.Items {
		if _, ok := j.namespaceToPoolName[job.Namespace]; !ok {
			continue
		}
		job := job
		j.jobUpdatedCallback(&job)
	}
	return nil
}

func (j *jobsService) refreshPodStates(allocationID model.AllocationID) error {
	if allocationID == "" {
		return fmt.Errorf("invalid call: allocationID missing")
	}

	pods, err := j.listPodsInAllNamespaces(context.TODO(), metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", determinedLabel, allocationID),
	})
	if err != nil {
		return errors.Wrap(err, "error listing pods checking if they can be restored")
	}

	for _, pod := range pods.Items {
		if _, ok := j.namespaceToPoolName[pod.Namespace]; !ok {
			continue
		}
		pod := pod
		j.podStatusCallback(&pod)
	}
	return nil
}

func (j *jobsService) deleteKubernetesResources(
	jobs *batchV1.JobList, configMaps *k8sV1.ConfigMapList,
) {
	for _, job := range jobs.Items {
		j.resourceRequestQueue.deleteKubernetesResources(job.Namespace, job.Name, "", "")
	}

	for _, configMap := range configMaps.Items {
		j.resourceRequestQueue.deleteKubernetesResources(configMap.Namespace, "", configMap.Name, "")
	}
}

func (j *jobsService) deleteDoomedKubernetesResources() error {
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
	jobs, err := j.listJobsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing pods")
	}

	toKillJobs := &batchV1.JobList{}
	savedJobNames := make(set.Set[string])
	for _, job := range jobs.Items {
		if _, ok := j.namespaceToPoolName[job.Namespace]; !ok {
			continue
		}

		resourcePool := job.Labels[resourcePoolLabel]
		if resourcePool == "" {
			j.syslog.Warnf("deleting job '%s' without resource pool label", job.Name)
			toKillJobs.Items = append(toKillJobs.Items, job)
			continue
		}

		allocationIDStr := job.Labels[determinedLabel]
		if allocationIDStr == "" {
			j.syslog.Warnf("deleting job '%s' without determined label (whose value is the allocation ID)", job.Name)
			toKillJobs.Items = append(toKillJobs.Items, job)
			continue
		}
		allocationID := model.AllocationID(allocationIDStr)

		if !openAllocationIDs.Contains(allocationID) {
			j.syslog.Warnf("deleting job '%s', did not find an open allocation for it", allocationID)
			toKillJobs.Items = append(toKillJobs.Items, job)
			continue
		}

		savedJobNames.Insert(job.Name)
	}

	configMaps, err := j.listConfigMapsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return errors.Wrap(err, "error listing existing config maps")
	}
	toKillConfigMaps := &k8sV1.ConfigMapList{}
	for _, cm := range configMaps.Items {
		if _, ok := j.namespaceToPoolName[cm.Namespace]; !ok {
			continue
		}

		if savedJobNames.Contains(cm.Name) { // Job name is same as config map name.
			continue
		}

		j.syslog.Debugf("deleting config map '%s', did not find a matching job that will be restored", cm.Name)
		toKillConfigMaps.Items = append(toKillConfigMaps.Items, cm)
	}

	j.deleteKubernetesResources(toKillJobs, toKillConfigMaps)
	return nil
}

func (j *jobsService) startNodeInformer() error {
	i, err := newNodeInformer(
		context.TODO(),
		j.clientSet.CoreV1().Nodes(),
		func(event watch.Event) {
			j.mu.Lock()
			defer j.mu.Unlock()
			j.nodeStatusCallback(event)
		})
	if err != nil {
		return err
	}

	go i.run(context.TODO())
	return nil
}

func (j *jobsService) startEventListeners() error {
	for namespace := range j.namespaceToPoolName {
		l, err := newEventInformer(
			context.TODO(),
			j.clientSet.CoreV1().Events(namespace),
			namespace,
			func(event watch.Event) {
				j.mu.Lock()
				defer j.mu.Unlock()
				j.newEventCallback(event)
			})
		if err != nil {
			return err
		}
		go l.run(context.TODO())
	}
	return nil
}

func (j *jobsService) startPreemptionListeners() error {
	for namespace := range j.namespaceToPoolName {
		l, err := newPodInformer(
			context.TODO(),
			determinedPreemptionLabel,
			"preemption",
			namespace,
			j.clientSet.CoreV1().Pods(namespace),
			func(event watch.Event) {
				j.mu.Lock()
				defer j.mu.Unlock()
				j.preemptionCallback(event)
			})
		if err != nil {
			return err
		}
		go l.run(context.TODO())
	}
	return nil
}

func (j *jobsService) startResourceRequestQueue() {
	failures := make(chan resourcesRequestFailure, 16)
	j.resourceRequestQueue = startRequestQueue(j.jobInterfaces, j.podInterfaces, j.configMapInterfaces, failures)
	j.wg.Go(func(ctx context.Context) {
		for {
			select {
			case failure := <-failures:
				j.handleResourceRequestFailure(failure)
			case <-ctx.Done():
				return
			}
		}
	})
}

func (j *jobsService) handleResourceRequestFailure(msg resourcesRequestFailure) {
	j.mu.Lock()
	defer j.mu.Unlock()

	jobName := msg.getJobName()
	jobHandler, ok := j.jobNameToJobHandler[jobName]
	if !ok {
		j.syslog.Warnf("received resource request error for unregistered pod %s", jobName)
		return
	}

	switch msg := msg.(type) {
	case resourceCreationFailed:
		jobHandler.receiveResourceCreationFailed(msg)
	case resourceCreationCancelled:
		jobHandler.receiveResourceCreationCancelled()
	case resourceDeletionFailed:
		jobHandler.receiveResourceDeletionFailed(msg)
	default:
		panic(fmt.Sprintf("unexpected message %T", msg))
	}

	err := j.cleanUpJobHandler(jobHandler)
	if err != nil {
		j.syslog.WithError(err).Error("cleaning up pod handler after resource request failure")
	}
}

func (j *jobsService) startJob(msg StartJob) error {
	newJobHandler := newJob(
		msg,
		msg.Spec.ClusterID,
		j.clientSet,
		msg.Namespace,
		j.detMasterIP,
		j.detMasterPort,
		j.masterTLSConfig,
		j.podInterfaces[msg.Namespace],
		j.configMapInterfaces[msg.Namespace],
		j.resourceRequestQueue,
		j.slotType,
		j.slotResourceRequests,
		j.scheduler,
	)

	if _, alreadyExists := j.jobNameToJobHandler[newJobHandler.jobName]; alreadyExists {
		return errors.Errorf(
			"attempting to register same job name: %s multiple times", newJobHandler.jobName)
	}

	err := newJobHandler.start()
	if err != nil {
		return fmt.Errorf("creating pod: %w", err)
	}

	j.jobNameToJobHandler[newJobHandler.jobName] = newJobHandler
	j.jobNameToResourcePool[newJobHandler.jobName] = msg.ResourcePool
	j.allocationIDToJobName[msg.Req.AllocationID] = newJobHandler.jobName
	j.jobNameToPodNameToSchedulingState[newJobHandler.jobName] = make(map[string]sproto.SchedulingState)
	j.jobHandlerToMetadata[newJobHandler] = jobMetadata{
		jobName:      newJobHandler.jobName,
		allocationID: newJobHandler.req.AllocationID,
	}

	return nil
}

func (j *jobsService) jobUpdatedCallback(obj any) {
	job, ok := obj.(*batchV1.Job)
	if !ok {
		j.syslog.Warnf("error converting event of type %T to *batchV1.Job: %+v", obj, obj)
		return
	}
	syslog := j.syslog.WithField("job", job.Name)

	jobHandler, ok := j.jobNameToJobHandler[job.Name]
	if !ok {
		syslog.Debugf("received job status update for un-registered job %s", job.Name)
		return
	}

	state, err := jobHandler.jobUpdatedCallback(job)
	if err != nil {
		syslog.WithError(err).Error("failed to process job status update")
		if err := j.cleanUpJobHandler(jobHandler); err != nil {
			syslog.WithError(err).Error("unable to cleanup job handler after an error")
		}
	} else if state == cproto.Terminated {
		if err := j.cleanUpJobHandler(jobHandler); err != nil {
			syslog.WithError(err).Error("unable to cleanup job handler after termination")
		}
	}
}

func (j *jobsService) jobDeletedCallback(obj any) {
	job, ok := obj.(*batchV1.Job)
	if !ok {
		j.syslog.Warnf("failed to convert event of type %T to *batchV1.Job: %+v", obj, obj)
		return
	}
	syslog := j.syslog.WithField("job", job.Name)

	jobHandler, ok := j.jobNameToJobHandler[job.Name]
	if !ok {
		syslog.Debugf("received job status update for un-registered job %s", job.Name)
		return
	}

	jobHandler.jobDeletedCallback()
	if err := j.cleanUpJobHandler(jobHandler); err != nil {
		syslog.WithError(err).Error("unable to cleanup job handler after an error")
	}
}

func (j *jobsService) podStatusCallback(obj any) {
	pod, ok := obj.(*k8sV1.Pod)
	if !ok {
		j.syslog.Warnf("error converting event of type %T to *k8sV1.Pod: %+v", obj, obj)
		return
	}
	syslog := j.syslog.WithField("pod", pod.Name)

	jobName, ok := pod.Labels[kubernetesJobNameLabel]
	if !ok {
		syslog.Debugf("received pod informer event for pod without %s label", kubernetesJobNameLabel)
		return
	}

	jobHandler, ok := j.jobNameToJobHandler[jobName]
	if !ok {
		syslog.Debugf("received pod status update for un-registered job %s", jobName)
		return
	}

	err := jobHandler.podUpdatedCallback(*pod)
	if err != nil {
		syslog.WithError(err).Error("error processing pod status update")
		return
	}

	j.updatePodSchedulingState(jobName, pod)
	if j.jobSchedulingStateCallback != nil {
		go j.jobSchedulingStateCallback(jobSchedulingStateChanged{
			AllocationID: jobHandler.req.AllocationID,
			NumPods:      jobHandler.numPods,
			State:        j.jobSchedulingState(jobName),
		})
	}
}

func (j *jobsService) podDeletedCallback(obj any) {
	pod, ok := obj.(*k8sV1.Pod)
	if !ok {
		j.syslog.Warnf("error converting event of type %T to *k8sV1.Pod: %+v", obj, obj)
		return
	}
	syslog := j.syslog.WithField("pod", pod.Name)

	jobName, ok := pod.Labels[kubernetesJobNameLabel]
	if !ok {
		syslog.Debugf("received pod informer event for pod without %s label", kubernetesJobNameLabel)
		return
	}

	jobHandler, ok := j.jobNameToJobHandler[jobName]
	if !ok {
		syslog.Debugf("received pod status update for un-registered job %s", jobName)
		return
	}

	jobHandler.podDeletedCallback(pod)
}

// jobSchedulingState is a roll-up of the sceduling states of its individual pods.
func (j *jobsService) jobSchedulingState(jobName string) sproto.SchedulingState {
	states, ok := j.jobNameToPodNameToSchedulingState[jobName]
	if !ok {
		return sproto.SchedulingStateQueued
	}
	if !allEqual(sproto.SchedulingStateScheduled, maps.Values(states)...) {
		return sproto.SchedulingStateQueued
	}
	return sproto.SchedulingStateScheduled
}

// updatePodSchedulingState stores the scheduling state of a pod based on its state (in particular the phase).
func (j *jobsService) updatePodSchedulingState(jobName string, pod *k8sV1.Pod) {
	states, ok := j.jobNameToPodNameToSchedulingState[jobName]
	if !ok {
		states = make(map[string]sproto.SchedulingState)
	}

	states[pod.Name] = sproto.SchedulingStateQueued
	if pod.Status.Phase == "Running" {
		states[pod.Name] = sproto.SchedulingStateScheduled
	}
	j.jobNameToPodNameToSchedulingState[jobName] = states
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

func (j *jobsService) enableNode(
	nodeName string,
) (*apiv1.EnableAgentResponse, error) {
	patch := []byte(fmt.Sprintf(`{
		"metadata": {
			"labels": {
				"%s": null
			}
		}
	}`, clusterIDNodeLabel()))

	_, err := j.clientSet.CoreV1().Nodes().
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
	j.syslog.Infof("node %s enabled by an user", nodeName)

	n, ok := j.summarizeClusterByNodes()[nodeName]
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

func (j *jobsService) disableNode(
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

	_, err = j.clientSet.CoreV1().Nodes().
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
	j.syslog.Infof("node %s disabled by a user", nodeName)

	if !shouldDrain { // See note in spec.go about how we could remove killing all pods here.
		if err := j.releaseAllocationsOnDisabledNode(nodeName); err != nil {
			return nil, fmt.Errorf(
				"node disabled without error, error killing existing pod on node: %w", err)
		}
	}

	n, ok := j.summarizeClusterByNodes()[nodeName]
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

func (j *jobsService) releaseAllocationsOnDisabledNode(nodeName string) error {
	listOptions := metaV1.ListOptions{
		LabelSelector: determinedLabel,
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	}
	pods, err := j.listPodsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return fmt.Errorf("listing pods on node %s: %w", nodeName, err)
	}

	notifiedAllocations := make(map[model.AllocationID]bool)
	for _, pod := range pods.Items {
		jobName, ok := pod.Labels[kubernetesJobNameLabel]
		if !ok {
			j.syslog.Debugf("found pod when disabling node without %s label", kubernetesJobNameLabel)
			continue
		}

		jobHandler, ok := j.jobNameToJobHandler[jobName]
		if !ok {
			j.syslog.Warnf(
				"during node disable couldn't find pod %s's actor to kill", pod.Name)
			continue
		}

		j.syslog.Infof(
			"stopping pod %s because node %s was disabled without drain option", pod.Name, nodeName)
		if notifiedAllocations[jobHandler.allocationID] {
			continue
		}

		rmevents.Publish(jobHandler.allocationID, &sproto.ReleaseResources{
			Reason:    "node disabled without drain",
			ForceKill: true,
		})
		notifiedAllocations[jobHandler.allocationID] = true
	}

	return nil
}

func (j *jobsService) nodeStatusCallback(event watch.Event) {
	node, ok := event.Object.(*k8sV1.Node)
	if !ok {
		j.syslog.Warnf("error converting event of type %T to *k8sV1.Node: %+v", event, event)
		return
	}

	j.syslog.Debugf(`informer got new node event for node '%s': %s %s`,
		node.Name, event.Type, node.Status.Phase)

	switch event.Type {
	case watch.Added:
		j.currentNodes[node.Name] = node
	case watch.Modified:
		j.currentNodes[node.Name] = node
	case watch.Deleted:
		delete(j.currentNodes, node.Name)
	default:
	}
}

func (j *jobsService) newEventCallback(event watch.Event) {
	newEvent, ok := event.Object.(*k8sV1.Event)
	if !ok {
		j.syslog.Warnf("error converting object type %T to *k8sV1.Event: %+v", event, event)
		return
	}
	syslog := j.syslog.WithFields(logrus.Fields{
		"name": newEvent.InvolvedObject.Name,
		"kind": newEvent.InvolvedObject.Kind,
	})

	switch newEvent.InvolvedObject.Kind {
	case "Pod": //nolint:goconst // Useless lint.
		podName := newEvent.InvolvedObject.Name
		jobNameParts := strings.Split(podName, "-")
		if len(jobNameParts) <= 1 {
			syslog.Tracef("received pod event for an un-registered pod %s", podName)
			return
		}
		jobName := strings.Join(jobNameParts[:len(jobNameParts)-1], "-")
		ref, ok := j.jobNameToJobHandler[jobName]
		if !ok {
			syslog.Tracef("received pod event for an un-registered job %s", jobName)
			return
		}
		ref.newEventCallback(newEvent)
	case "Job":
		jobName := newEvent.InvolvedObject.Name
		ref, ok := j.jobNameToJobHandler[jobName]
		if !ok {
			syslog.Tracef("received job event for an un-registered job %s", jobName)
			return
		}
		ref.newEventCallback(newEvent)
	}
}

func (j *jobsService) receiveResourceSummarize(msg SummarizeResources) (*PodsInfo, error) {
	summary, err := j.summarize()
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

func (j *jobsService) preemptionCallback(event watch.Event) {
	pod, ok := event.Object.(*k8sV1.Pod)
	if !ok {
		j.syslog.Warnf("error converting event of type %T to *k8sV1.Pod: %+v", event, event)
		return
	}
	j.syslog.Debugf("informer got new preemption event for pod %s ", pod.Name)

	ref, ok := j.jobNameToJobHandler[pod.Name]
	if !ok {
		j.syslog.Debug("received preemption command for unregistered pod")
		return
	}
	ref.preemptionCallback()
}

func (j *jobsService) verifyJobAndGetRef(id model.AllocationID) (*job, error) {
	jobName, ok := j.allocationIDToJobName[id]
	if !ok {
		return nil, fmt.Errorf("unknown allocation %s", id)
	}

	ref, ok := j.jobNameToJobHandler[jobName]
	if !ok {
		return nil, fmt.Errorf("unknown job %s", jobName)
	}
	return ref, nil
}

func (j *jobsService) receivePriorityChange(id model.AllocationID) {
	ref, err := j.verifyJobAndGetRef(id)
	if err != nil {
		j.syslog.WithError(err).Debug("changing allocation priority")
		return
	}
	ref.changePriority()
}

func (j *jobsService) receivePositionChange(id model.AllocationID) {
	ref, err := j.verifyJobAndGetRef(id)
	if err != nil {
		j.syslog.WithError(err).Debug("changing allocation position")
		return
	}
	ref.changePosition()
}

func (j *jobsService) receiveKill(id model.AllocationID) {
	ref, err := j.verifyJobAndGetRef(id)
	if err != nil {
		j.syslog.WithError(err).Debug("killing allocation")
		return
	}
	ref.Kill()
}

func (j *jobsService) cleanUpJobHandler(jobHandler *job) error {
	jobHandler.finalize()

	jobInfo, ok := j.jobHandlerToMetadata[jobHandler]
	if !ok {
		return errors.Errorf("unknown job handler being deleted %s", jobHandler.jobName)
	}

	j.syslog.
		WithField("pod", jobInfo.jobName).
		WithField("handler", jobHandler.jobName).
		Infof("de-registering job handler")
	delete(j.jobNameToJobHandler, jobInfo.jobName)
	delete(j.jobNameToResourcePool, jobInfo.jobName)
	delete(j.allocationIDToJobName, jobInfo.allocationID)
	delete(j.jobNameToPodNameToSchedulingState, jobInfo.jobName)
	delete(j.jobHandlerToMetadata, jobHandler)

	// launch this work async, since we hold the lock and it does API calls.
	j.wg.Go(func(ctx context.Context) {
		name := fmt.Sprintf("%s-priorityclass", jobInfo.allocationID)
		err := j.clientSet.
			SchedulingV1().
			PriorityClasses().
			Delete(ctx, name, metaV1.DeleteOptions{})
		if err != nil && !k8error.IsNotFound(err) {
			j.syslog.Warnf("Deletion of PriorityClass %s failed.", name)
		}
	})

	return nil
}

func (j *jobsService) handleGetSlotsRequest(agentID string) *apiv1.GetSlotsResponse {
	agentResp := j.handleGetAgentRequest(agentID)
	if agentResp == nil {
		j.syslog.Warnf("no agent with id %s", agentID)
		return nil
	}
	return &apiv1.GetSlotsResponse{Slots: maps.Values(agentResp.Agent.Slots)}
}

func (j *jobsService) handleGetSlotRequest(agentID string, slotID string) *apiv1.GetSlotResponse {
	agentResp := j.handleGetAgentRequest(agentID)
	if agentResp == nil {
		j.syslog.Warnf("no agent with id %s", agentID)
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
			j.syslog.Warnf("no slot with id %s", slotID)
			return nil
		}
	}
	return &apiv1.GetSlotResponse{Slot: slot}
}

func (j *jobsService) handleGetAgentsRequest() *apiv1.GetAgentsResponse {
	j.getAgentsCacheLock.Lock()
	defer j.getAgentsCacheLock.Unlock()

	if time.Since(j.getAgentsCacheTime) > getAgentsCacheDuration {
		j.getAgentsCacheTime = time.Now()

		nodeSummaries := j.summarizeClusterByNodes()
		_, nodesToPools := j.getNodeResourcePoolMapping(nodeSummaries)

		j.getAgentsCache = &apiv1.GetAgentsResponse{}
		for _, summary := range nodeSummaries {
			summary.ResourcePool = nodesToPools[summary.ID]
			j.getAgentsCache.Agents = append(j.getAgentsCache.Agents, summary.ToProto())
		}
	}

	return j.getAgentsCache
}

func (j *jobsService) handleGetAgentRequest(agentID string) *apiv1.GetAgentResponse {
	nodeSummaries := j.summarizeClusterByNodes()
	_, nodesToPools := j.getNodeResourcePoolMapping(nodeSummaries)
	agentSummary, ok := nodeSummaries[agentID]
	if !ok {
		// TODO(DET-10029): We should return an error indicating the invalid ID request (rather
		//	than a warn).
		j.syslog.Warnf("no agent with id %s", agentID)
		return nil
	}
	agentSummary.ResourcePool = nodesToPools[agentSummary.ID]
	return &apiv1.GetAgentResponse{Agent: agentSummary.ToProto()}
}

// summarize describes pods' available resources. When there's exactly one resource pool, it uses
// the whole cluster's info. Otherwise, it matches nodes to resource pools using taints and
// tolerations to derive that info. This may be cached, so don't use this for decisions
// that require up-to-date information.
func (j *jobsService) summarize() (map[string]model.AgentSummary, error) {
	j.summarizeCacheLock.Lock()
	defer j.summarizeCacheLock.Unlock()

	if time.Since(j.summarizeCacheTime) > summarizeCacheDuration {
		summary, err := j.computeSummary()
		j.summarizeCacheTime = time.Now()
		j.summarizeCache = summarizeResult{
			summary: summary,
			err:     err,
		}
	}

	return j.summarizeCache.summary, j.summarizeCache.err
}

// Get the mapping of many-to-many relationship between nodes and resource pools.
func (j *jobsService) getNodeResourcePoolMapping(nodeSummaries map[string]model.AgentSummary) (
	map[string][]*k8sV1.Node, map[string][]string,
) {
	poolTaskContainerDefaults := extractTCDs(j.resourcePoolConfigs)

	// Nvidia automatically taints nodes, so we should tolerate that when users don't customize
	// their resource pool config.
	defaultTolerations := []k8sV1.Toleration{{
		Key:      ResourceTypeNvidia,
		Value:    "present",
		Operator: k8sV1.TolerationOpEqual,
	}}
	cpuTolerations, gpuTolerations := extractTolerations(j.baseContainerDefaults)
	poolsToNodes := make(map[string][]*k8sV1.Node, len(j.namespaceToPoolName))
	nodesToPools := make(map[string][]string, len(j.namespaceToPoolName))

	for _, node := range j.currentNodes {
		_, slotType := extractSlotInfo(nodeSummaries[node.Name])

		for poolName, tcd := range poolTaskContainerDefaults {
			var poolTolerations []k8sV1.Toleration

			// If they're using the default RP config, use the default tolerations.
			if len(j.resourcePoolConfigs) <= 1 &&
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

func (j *jobsService) computeSummary() (map[string]model.AgentSummary, error) {
	nodeSummaries := j.summarizeClusterByNodes()

	// Build the many-to-many relationship between nodes and resource pools
	poolsToNodes, _ := j.getNodeResourcePoolMapping(nodeSummaries)

	// Build the set of summaries for each resource pool
	containers := j.containersPerResourcePool()
	summaries := make(map[string]model.AgentSummary, len(j.namespaceToPoolName))
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

func (j *jobsService) summarizeClusterByNodes() map[string]model.AgentSummary {
	var allPods []podNodeInfo

	for _, p := range j.jobNameToJobHandler {
		allPods = append(allPods, p.getNodeInfoForPods()...)
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

	nodeToTasks, taskSlots := j.getNonDetSlots(j.slotType)
	summary := make(map[string]model.AgentSummary, len(j.currentNodes))
	for _, node := range j.currentNodes {
		disabledLabel, isDisabled := node.Labels[clusterIDNodeLabel()]
		isDraining := isDisabled && disabledLabel == noScheduleNodeLabelValue

		var numSlots int64
		var deviceType device.Type

		// TODO(DET-10010): slot type per node probably shouldn't be decided from pods literal
		// (which has the same value for all nodes).
		switch j.slotType {
		case device.CPU:
			resources := node.Status.Allocatable[k8sV1.ResourceCPU]
			milliCPUs := resources.MilliValue() - j.nodeToSystemResourceRequests[node.Name]
			numSlots = int64(float32(milliCPUs) / (1000. * j.slotResourceRequests.CPU))
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
					j.syslog.Warnf("too many pods mapping to node %s", node.Name)
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
					j.syslog.Warnf("too many pods mapping to node %s", node.Name)
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

func (j *jobsService) getNonDetPods() ([]k8sV1.Pod, error) {
	// TODO(RM-235) use a filter in metaV1.ListOptions. This change gets a lot easier after
	// we have K8s integration tests. Using a filter means we should really talk to a real
	// k8s server. Doing an e2e test for this is possible but would take a lot more work.
	allPods, err := j.listPodsInAllNamespaces(context.TODO(), metaV1.ListOptions{})
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

func (j *jobsService) getNonDetSlots(deviceType device.Type) (map[string][]string, map[string]int64) {
	nodeToTasks := make(map[string][]string, len(j.currentNodes))
	taskSlots := make(map[string]int64)

	nonDetPods, err := j.getNonDetPods()
	if err != nil {
		j.syslog.WithError(err).Warn("getting non determined pods, " +
			"this may cause slots to look free when they are in use")
	}

	if len(nonDetPods) == 0 {
		return nodeToTasks, taskSlots
	}
	for _, node := range j.currentNodes {
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
				reqs += j.getCPUReqs(c)
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

func (j *jobsService) getCPUReqs(c k8sV1.Container) int64 {
	requested := float32(c.Resources.Requests.Cpu().MilliValue()) /
		(1000. * j.slotResourceRequests.CPU)
	return int64(requested)
}

func (j *jobsService) containersPerResourcePool() map[string]int {
	counts := make(map[string]int, len(j.namespaceToPoolName))
	for name, pool := range j.jobNameToResourcePool {
		handler, ok := j.jobNameToJobHandler[name]
		if !ok {
			j.syslog.Errorf("job %s not in jobNameToResourcePool but in jobNameToJobHandler map", name)
			continue
		}
		counts[pool] += handler.numPods
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

func (j *jobsService) listJobsInAllNamespaces(
	ctx context.Context, opts metaV1.ListOptions,
) (*batchV1.JobList, error) {
	res := &batchV1.JobList{}
	for n, i := range j.jobInterfaces {
		pods, err := i.List(ctx, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "error listing pods for namespace %s", n)
		}

		res.Items = append(res.Items, pods.Items...)
	}

	return res, nil
}

func (j *jobsService) listPodsInAllNamespaces(
	ctx context.Context, opts metaV1.ListOptions,
) (*k8sV1.PodList, error) {
	res := &k8sV1.PodList{}
	for n, i := range j.podInterfaces {
		pods, err := i.List(ctx, opts)
		if err != nil {
			return nil, errors.Wrapf(err, "error listing pods for namespace %s", n)
		}

		res.Items = append(res.Items, pods.Items...)
	}

	return res, nil
}

func (j *jobsService) listConfigMapsInAllNamespaces(
	ctx context.Context, opts metaV1.ListOptions,
) (*k8sV1.ConfigMapList, error) {
	res := &k8sV1.ConfigMapList{}
	for n, i := range j.configMapInterfaces {
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

func all[T any](pred func(T) bool, elems ...T) bool {
	for _, elem := range elems {
		if !pred(elem) {
			return false
		}
	}
	return true
}

func allEqual[T comparable](other T, elems ...T) bool {
	return all(func(elem T) bool {
		return elem == other
	}, elems...)
}
