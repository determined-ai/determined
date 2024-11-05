package kubernetesrm

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"
	batchV1 "k8s.io/api/batch/v1"
	k8sV1 "k8s.io/api/core/v1"
	k8error "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/informers"
	k8sClient "k8s.io/client-go/kubernetes"
	typedBatchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
	alphaGatewayTyped "sigs.k8s.io/gateway-api/apis/v1alpha2"
	gateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1"
	alphaGateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1alpha2"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/rm"
	"github.com/determined-ai/determined/master/internal/rm/rmevents"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/logger"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/set"
	"github.com/determined-ai/determined/master/pkg/syncx/waitgroupx"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

const (
	determinedLabel           = "determined"
	determinedPreemptionLabel = "determined-preemption"
	determinedSystemLabel     = "determined-system"
	jobNameAnnotation         = "determined.ai/job-name"

	kubernetesJobNameLabel = "batch.kubernetes.io/job-name"

	resourceTypeNvidia = "nvidia.com/gpu"
	defaultNamespace   = "default"
	// ReleaseNamespaceEnvVar is the name of the environment variable within a pod running the
	// master service containing the namespace in which determined was deployed.
	ReleaseNamespaceEnvVar = "DET_RELEASE_NAMESPACE"
	// ResourceTypeNvidia describes the GPU resource type.
	ResourceTypeNvidia = "nvidia.com/gpu"
)

var cacheSyncs []cache.InformerSynced

type summarizeResult struct {
	summary map[string]model.AgentSummary
	err     error
}

type jobMetadata struct {
	jobName      string
	allocationID model.AllocationID
}

// High lever overview of the actors within the kubernetes package:
//
//	jobsService
//	  +- job(s): manages pod lifecycle. One per container in a task.
//	     +- podLogStreamer: stream logs for a specific pod.
//	  +- informer: sends updates about pod states
//	  +- events: sends updates about kubernetes events.
//	  +- requestQueue: queues requests to create / delete kubernetes resources.
//	     +- requestProcessingWorkers: processes request to create / delete kubernetes resources.
type jobsService struct {
	// Configuration details. Set in initialization (the `newJobService` constructor) and never modified after.
	namespace             string
	clusterName           string
	scheduler             string
	slotType              device.Type
	slotResourceRequests  config.PodSlotResourceRequests
	resourcePoolConfigs   []config.ResourcePoolConfig
	baseContainerDefaults *model.TaskContainerDefaultsConfig
	masterServiceName     string
	masterTLSConfig       model.TLSClientConfig
	detMasterIP           string
	detMasterPort         int32
	detMasterScheme       string
	kubeconfigPath        string

	internalTaskGWConfig *config.InternalTaskGatewayConfig

	// System dependencies. Also set in initialization and never modified after.
	syslog    *logrus.Entry
	clientSet k8sClient.Interface
	// TODO(!!!): Not set in initialization and never changed anymore.. RIP.
	podInterfaces       map[string]typedV1.PodInterface
	configMapInterfaces map[string]typedV1.ConfigMapInterface
	jobInterfaces       map[string]typedBatchV1.JobInterface
	serviceInterfaces   map[string]typedV1.ServiceInterface
	tcpRouteInterfaces  map[string]alphaGateway.TCPRouteInterface
	// TODO(!!!): end.

	resourceRequestQueue       *requestQueue
	requestQueueWorkers        []*requestProcessingWorker
	jobSchedulingStateCallback jobSchedulingStateCallback

	// Internal state. Access should be protected.
	wg                                waitgroupx.Group
	mu                                sync.RWMutex
	jobNameToJobHandler               map[string]*job
	jobNameToResourcePool             map[string]string
	jobNameToPodNameToSchedulingState map[string]map[string]sproto.SchedulingState
	allocationIDToJobName             map[model.AllocationID]string
	jobHandlerToMetadata              map[*job]jobMetadata
	nodeToSystemResourceRequests      map[string]int64
	currentNodes                      map[string]*k8sV1.Node
	gatewayService                    *gatewayService

	// TODO(RM-236) make one cache and make this code more straightforward.
	summarizeCacheLock      sync.RWMutex
	summarizeCache          summarizeResult
	summarizeCacheTime      time.Time
	getAgentsCacheLock      sync.Mutex
	getAgentsCache          *apiv1.GetAgentsResponse
	getAgentsCacheTime      time.Time
	namespacesWithInformers map[string]bool
}

func (j *jobsService) GetAllNamespacesForRM() ([]string, error) {
	ns, err := workspace.GetAllNamespacesForRM(context.Background(), j.clusterName)
	if err != nil {
		return ns, err
	}
	if j.namespace == "" {
		j.namespace = defaultNamespace
	}
	if !slices.Contains(ns, j.namespace) {
		ns = append(ns, j.namespace)
	}
	return ns, nil
}

// newJobsService creates a new pod service for launching, querying and interacting with k8s pods.
func newJobsService(
	namespace string,
	clusterName string,
	masterServiceName string,
	masterTLSConfig model.TLSClientConfig,
	scheduler string,
	slotType device.Type,
	slotResourceRequests config.PodSlotResourceRequests,
	resourcePoolConfigs []config.ResourcePoolConfig,
	taskContainerDefaults *model.TaskContainerDefaultsConfig,
	detMasterIP string,
	detMasterPort int32,
	detMasterScheme string,
	kubeconfigPath string,
	jobSchedulingStateCb jobSchedulingStateCallback,
	internalTaskGWConfig *config.InternalTaskGatewayConfig,
) (*jobsService, error) {
	p := &jobsService{
		wg: waitgroupx.WithContext(context.Background()),

		namespace:                         namespace,
		clusterName:                       clusterName,
		masterServiceName:                 masterServiceName,
		masterTLSConfig:                   masterTLSConfig,
		detMasterScheme:                   detMasterScheme,
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
		serviceInterfaces:                 make(map[string]typedV1.ServiceInterface),
		tcpRouteInterfaces:                make(map[string]alphaGateway.TCPRouteInterface),
		syslog:                            logrus.WithField("namespace", namespace),
		jobSchedulingStateCallback:        jobSchedulingStateCb,

		internalTaskGWConfig:    internalTaskGWConfig,
		kubeconfigPath:          kubeconfigPath,
		namespacesWithInformers: make(map[string]bool),
	}

	ns, err := p.GetAllNamespacesForRM()
	if err != nil {
		panic(fmt.Errorf("failed to get namespaces for resource manager: %w", err))
	}

	if err := p.startClientSet(ns); err != nil {
		return nil, err
	}
	if err := p.getMasterIPAndPort(); err != nil {
		return nil, err
	}
	if err := p.getSystemResourceRequests(); err != nil {
		return nil, err
	}

	p.startResourceRequestQueue()

	if err := p.deleteDoomedKubernetesResources(ns); err != nil {
		return nil, err
	}

	err = p.startNodeInformer()
	switch {
	case err != nil && k8error.IsForbidden(err):
		p.syslog.Warnf("unable to start node informer due to permission error,"+
			"some features will be degraded: %s", err,
		)
	case err != nil:
		return nil, err
	}

	err = p.syncNamespaces(ns, false)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (j *jobsService) syncNamespaces(ns []string, hasJSLock bool) error {
	// TODO(!!!): Prob one informer per cluster too.
	for _, namespace := range ns {
		// Since we don't want to do duplicate namespace informers, don't start any
		// listeners or informers that have already been added to namespacesWithInformers.
		if _, ok := j.namespacesWithInformers[namespace]; ok {
			continue
		}

		err := j.startEventListeners(namespace, hasJSLock)
		if err != nil {
			return err
		}

		// Once we have started event listeners for a namespace, track these synced namespaces in
		// namespacesWithInformers.
		j.namespacesWithInformers[namespace] = true

		err = j.startPreemptionListeners(namespace, hasJSLock)
		if err != nil {
			return err
		}

		factory := informers.NewSharedInformerFactoryWithOptions(j.clientSet, time.Hour,
			informers.WithNamespace(namespace))

		jobsInformer := factory.Batch().V1().Jobs()
		if _, err := jobsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				j.mu.Lock()
				defer j.mu.Unlock()
				j.jobUpdatedCallback(obj)
			},
			UpdateFunc: func(_, obj interface{}) {
				j.mu.Lock()
				defer j.mu.Unlock()
				j.jobUpdatedCallback(obj)
			},

			// If a job is deleted out from under us, this is the only hook we have to not
			// leave our workloads running or pending forever.
			DeleteFunc: func(obj interface{}) {
				j.mu.Lock()
				defer j.mu.Unlock()
				j.jobDeletedCallback(obj)
			},
		}); err != nil {
			return fmt.Errorf("adding job informer: %w", err)
		}

		cacheSyncs = append(cacheSyncs, jobsInformer.Informer().HasSynced)

		podsInformer := factory.Core().V1().Pods()
		if _, err := podsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				j.mu.Lock()
				defer j.mu.Unlock()
				j.podStatusCallback(obj)
			},
			UpdateFunc: func(_, obj interface{}) {
				j.mu.Lock()
				defer j.mu.Unlock()
				j.podStatusCallback(obj)
			},

			// If a pod is deleted out from under us, it is nice to let the user know that
			// is what happened.
			DeleteFunc: func(obj interface{}) {
				j.mu.Lock()
				defer j.mu.Unlock()
				j.podDeletedCallback(obj)
			},
		}); err != nil {
			return fmt.Errorf("adding pod informer: %w", err)
		}
		cacheSyncs = append(cacheSyncs, podsInformer.Informer().HasSynced)

		factory.Start(nil)
	}

	if !cache.WaitForCacheSync(nil, cacheSyncs...) {
		return errors.New("failed to wait for cache sync for jobs informer")
	}
	return nil
}

func (j *jobsService) startClientSet(namespaces []string) error {
	config, err := readClientConfig(j.kubeconfigPath)
	if err != nil {
		return fmt.Errorf("error building kubernetes config: %w", err)
	}

	j.clientSet, err = k8sClient.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to initialize kubernetes clientSet: %w", err)
	}

	j.jobInterfaces[""] = j.clientSet.BatchV1().Jobs("")
	j.podInterfaces[""] = j.clientSet.CoreV1().Pods("")
	for _, ns := range namespaces {
		j.podInterfaces[ns] = j.clientSet.CoreV1().Pods(ns)
		j.configMapInterfaces[ns] = j.clientSet.CoreV1().ConfigMaps(ns)
		j.jobInterfaces[ns] = j.clientSet.BatchV1().Jobs(ns)
	}

	if taskGWConfig := j.internalTaskGWConfig; taskGWConfig != nil {
		// Using the CoreV1 RESTClient for gateway resources will cause "resource not found" errors.
		alphaGatewayClientSet, err := alphaGateway.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("creating Kubernetes gateway clientSet: %w", err)
		}
		for _, ns := range namespaces {
			j.serviceInterfaces[ns] = j.clientSet.CoreV1().Services(ns)
			j.tcpRouteInterfaces[ns] = alphaGatewayClientSet.TCPRoutes(ns)
		}

		// Using the alphaGateway clientSet will not work properly.
		gatewayClientSet, err := gateway.NewForConfig(config)
		if err != nil {
			return fmt.Errorf("creating Kubernetes gateway clientSet: %w", err)
		}
		gwService, err := newGatewayService(
			gatewayClientSet.Gateways(taskGWConfig.GatewayNamespace),
			j.tcpRouteInterfaces,
			*taskGWConfig,
		)
		if err != nil {
			return fmt.Errorf("creating gateway service: %w", err)
		}
		j.gatewayService = gwService
	}

	j.syslog.Infof("kubernetes clientSet initialized")
	return nil
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
		c, err := rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
		if c.QPS == 0.0 {
			c.QPS = 20
		}
		if c.Burst == 0 {
			c.Burst = 100
		}
		return c, nil
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

func (j *jobsService) getMasterIPAndPort() error {
	if j.detMasterIP != "" && j.detMasterPort != 0 {
		// Master ip and port were manually configured. For special circumstances, e.g., the master is running
		// outside of this cluster (happens in development or when we spread across multiple k8s clusters).
		return nil
	}
	masterService, err := j.clientSet.CoreV1().
		Services(j.getInitialNamespace()).
		Get(context.TODO(), j.masterServiceName, metaV1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get master service: %w", err)
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
		return fmt.Errorf("failed to get system pods: %w", err)
	}

	for _, systemPod := range systemPods.Items {
		for _, container := range systemPod.Spec.Containers {
			j.nodeToSystemResourceRequests[systemPod.Spec.NodeName] += container.Resources.Requests.Cpu().
				MilliValue()
		}
	}
	return nil
}

func (j *jobsService) deleteDoomedKubernetesResources(namespaces []string) error {
	var openAllocations []model.Allocation
	if err := db.Bun().NewSelect().Model(&openAllocations).
		Where("end_time IS NULL").
		Scan(context.TODO()); err != nil {
		return fmt.Errorf("error querying the database for open allocations: %w", err)
	}
	openAllocationIDs := make(set.Set[model.AllocationID])
	for _, alloc := range openAllocations {
		openAllocationIDs.Insert(alloc.AllocationID)
	}
	j.syslog.Infof("found open allocations %s", openAllocationIDs)

	listOptions := metaV1.ListOptions{LabelSelector: determinedLabel}
	jobs, err := j.listJobsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return fmt.Errorf("error listing existing pods: %w", err)
	}

	var toKillJobs []batchV1.Job
	savedJobNames := make(set.Set[string])
	for _, job := range jobs {
		if !slices.Contains(namespaces, job.Namespace) {
			continue
		}

		resourcePool := job.Labels[resourcePoolLabel]
		if resourcePool == "" {
			j.syslog.Warnf("deleting job '%s' without resource pool label", job.Name)
			toKillJobs = append(toKillJobs, job)
			continue
		}

		allocationIDStr := job.Labels[allocationIDLabel]
		if allocationIDStr == "" {
			j.syslog.Warnf("deleting job '%s' without determined label (whose value is the allocation ID)", job.Name)
			toKillJobs = append(toKillJobs, job)
			continue
		}
		allocationID := model.AllocationID(allocationIDStr)

		if !openAllocationIDs.Contains(allocationID) {
			j.syslog.
				WithField("allocation-id", allocationID).
				Warnf("deleting job '%s', did not find an open allocation for it", job.Name)
			toKillJobs = append(toKillJobs, job)
			continue
		}

		savedJobNames.Insert(job.Name)
	}

	resourceIsSaved := func(namespace, jobName string) bool {
		if !slices.Contains(namespaces, namespace) {
			return true
		}

		return savedJobNames.Contains(jobName)
	}
	configMaps, err := j.listConfigMapsInAllNamespaces(context.TODO(), listOptions)
	if err != nil {
		return fmt.Errorf("error listing existing config maps: %w", err)
	}
	var toKillConfigMaps []k8sV1.ConfigMap
	for _, cm := range configMaps {
		// Config map name and job name are the same.
		if resourceIsSaved(cm.Namespace, cm.Name) {
			continue
		}

		j.syslog.Debugf("deleting config map '%s', did not find a matching job that will be restored", cm.Name)
		toKillConfigMaps = append(toKillConfigMaps, cm)
	}

	var toKillServices []k8sV1.Service
	var toKillTCPRoutes []alphaGatewayTyped.TCPRoute
	var toFreeGatewayPorts []int
	if j.internalTaskGWConfig != nil {
		services, err := j.listServicesInAllNamespaces(context.TODO(), listOptions)
		if err != nil {
			return fmt.Errorf("listing existing services: %w", err)
		}
		for _, s := range services {
			if resourceIsSaved(s.Namespace, s.Annotations[jobNameAnnotation]) {
				continue
			}

			j.syslog.Debugf("deleting service '%s', did not find a matching job that will be restored", s.Name)
			toKillServices = append(toKillServices, s)
		}

		savedGatewayPorts := make(map[int]bool)
		tcpRoutes, err := j.listTCPRoutesInAllNamespaces(context.TODO(), listOptions)
		if err != nil {
			return fmt.Errorf("listing existing services: %w", err)
		}
		for _, t := range tcpRoutes {
			if resourceIsSaved(t.Namespace, t.Annotations[jobNameAnnotation]) {
				for _, s := range t.Spec.ParentRefs {
					if p := s.Port; p != nil {
						savedGatewayPorts[int(*p)] = true
					}
				}

				continue
			}

			j.syslog.Debugf("deleting TCPRoute '%s', did not find a matching job that will be restored", t.Name)
			toKillTCPRoutes = append(toKillTCPRoutes, t)
		}

		gatewayPorts, err := j.gatewayService.getProxyPorts(nil)
		if err != nil {
			return fmt.Errorf("listing gateway ports: %w", err)
		}
		for _, p := range gatewayPorts {
			if savedGatewayPorts[p] {
				continue
			}

			j.syslog.Debugf("freeing Gateway port '%d', did not find a matching job that will be restored", p)
			toFreeGatewayPorts = append(toFreeGatewayPorts, p)
		}
	}

	j.deleteKubernetesResources(
		toKillJobs,
		toKillConfigMaps,
		toKillServices,
		toKillTCPRoutes,
		toFreeGatewayPorts,
	)
	return nil
}

// startJob notifies the pods actor to start a pod with the task spec.
type startJob struct {
	req          *sproto.AllocateRequest
	allocationID model.AllocationID
	spec         tasks.TaskSpec
	slots        int
	rank         int
	resourcePool string
	namespace    string

	numPods int

	logContext logger.Context
}

func (j *jobsService) StartJob(msg startJob) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.startJob(msg)
}

func (j *jobsService) startJob(msg startJob) error {
	newJobHandler := newJob(
		configureUniqueName(msg.spec),
		msg,
		msg.spec.ClusterID,
		j.clientSet,
		msg.namespace,
		j.detMasterIP,
		j.detMasterPort,
		j.detMasterScheme,
		j.masterTLSConfig,
		j.podInterfaces[msg.namespace],
		j.configMapInterfaces[msg.namespace],
		j.resourceRequestQueue,
		j.slotType,
		j.slotResourceRequests,
		j.scheduler,
		j.internalTaskGWConfig,
		j.gatewayService,
	)

	if _, alreadyExists := j.jobNameToJobHandler[newJobHandler.jobName]; alreadyExists {
		return fmt.Errorf("attempting to register same job name: %s multiple times", newJobHandler.jobName)
	}

	err := newJobHandler.createSpecAndSubmit(&msg.spec)
	if err != nil {
		return fmt.Errorf("creating pod: %w", err)
	}

	j.jobNameToJobHandler[newJobHandler.jobName] = newJobHandler
	j.jobNameToResourcePool[newJobHandler.jobName] = msg.resourcePool
	j.allocationIDToJobName[msg.req.AllocationID] = newJobHandler.jobName
	j.jobNameToPodNameToSchedulingState[newJobHandler.jobName] = make(map[string]sproto.SchedulingState)
	j.jobHandlerToMetadata[newJobHandler] = jobMetadata{
		jobName:      newJobHandler.jobName,
		allocationID: newJobHandler.req.AllocationID,
	}

	return nil
}

func (j *jobsService) ChangePriority(id model.AllocationID) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.changePriority(id)
}

func (j *jobsService) KillJob(id model.AllocationID) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.killJob(id)
}

func (j *jobsService) SummarizeResources(poolName string) (*computeUsageSummary, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.summarizeComputeUsage(poolName)
}

func (j *jobsService) ReattachJob(msg reattachJobRequest) (reattachJobResponse, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.reattachJob(msg)
}

func (j *jobsService) DefaultNamespace() string {
	j.mu.Lock()
	defer j.mu.Unlock()

	return j.namespace
}

func (j *jobsService) VerifyNamespaceExists(namespace string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.verifyNamespaceExists(namespace, true)
}

func (j *jobsService) CreateNamespace(namespace string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.createNamespace(namespace, true)
}

func (j *jobsService) DeleteNamespace(namespace string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.deleteNamespace(namespace)
}

func (j *jobsService) RemoveEmptyNamespace(namespaceName string, clusterName string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.removeEmptyNamespace(namespaceName, clusterName)
}

func (j *jobsService) SetResourceQuota(quota int, namespace string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.setResourceQuota(quota, namespace)
}

func (j *jobsService) GetNamespaceResourceQuota(namespaceName string) (*float64, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.getNamespaceResourceQuota(namespaceName)
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

func (j *jobsService) reattachJob(msg reattachJobRequest) (reattachJobResponse, error) {
	// Get all expected resources for the job.
	listOptions := metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", determinedLabel, msg.allocationID),
	}

	var errs *multierror.Error
	jobs, err := j.listJobsInAllNamespaces(context.TODO(), listOptions)
	errs = multierror.Append(errs, err)

	configMaps, err := j.listConfigMapsInAllNamespaces(context.TODO(), listOptions)
	errs = multierror.Append(errs, err)

	var services []k8sV1.Service
	var tcpRoutes []alphaGatewayTyped.TCPRoute
	var gatewayPorts []int
	if j.internalTaskGWConfig != nil {
		services, err = j.listServicesInAllNamespaces(context.TODO(), listOptions)
		errs = multierror.Append(errs, err)

		tcpRoutes, err = j.listTCPRoutesInAllNamespaces(context.TODO(), listOptions)
		errs = multierror.Append(errs, err)

		gatewayPorts, err = j.gatewayService.getProxyPorts(&msg.allocationID)
		errs = multierror.Append(errs, err)
	}

	// Do a sanity check validate. Is this a job that can reattach?
	// Err on the side of caution here.
	if len(jobs) != 1 {
		errs = multierror.Append(errs, fmt.Errorf("expected one job got %d", len(jobs)))
	}
	if len(configMaps) != 1 {
		errs = multierror.Append(errs, fmt.Errorf("expected one config map got %d", len(configMaps)))
	}
	expectedProxyNum := len(msg.req.ProxyPorts)
	if j.internalTaskGWConfig != nil && expectedProxyNum > 0 {
		if len(services) != expectedProxyNum {
			errs = multierror.Append(errs,
				fmt.Errorf("expected %d services got %d", expectedProxyNum, len(services)))
		}
		if len(tcpRoutes) != expectedProxyNum {
			errs = multierror.Append(errs,
				fmt.Errorf("expected %d tcpRoutes got %d", expectedProxyNum, len(services)))
		}
		if len(gatewayPorts) != expectedProxyNum {
			errs = multierror.Append(errs,
				fmt.Errorf("expected %d gateway ports got %d", expectedProxyNum, len(gatewayPorts)))
		}
	}

	// Cleanup the job if we don't get the format we expect.
	cleanup := func() {
		j.deleteKubernetesResources(jobs, configMaps, services, tcpRoutes, gatewayPorts)
	}
	if errs.Len() > 0 {
		cleanup()
		return reattachJobResponse{}, fmt.Errorf("reattach job: %w", errs)
	}

	job := jobs[0]
	if len(jobs) != 1 { // Unnecessary, but we should be careful here.
		cleanup()
		return reattachJobResponse{}, fmt.Errorf("expected one job")
	}

	resourcePool, ok := job.Labels[resourcePoolLabel]
	if !ok {
		cleanup()
		return reattachJobResponse{}, fmt.Errorf("could not recover resource pool for %s", msg.allocationID)
	}

	gatewayResources, err := j.recreateGatewayProxyResources(
		services, tcpRoutes, gatewayPorts,
	)
	if err != nil {
		cleanup()
		return reattachJobResponse{}, err
	}

	resp, err := j.recreateJobHandler(
		job.Name,
		msg.req,
		msg.allocationID,
		resourcePool,
		&job,
		msg.slots,
		msg.numPods,
		gatewayResources,
		msg.logContext,
	)
	if err != nil {
		cleanup()
		return reattachJobResponse{}, fmt.Errorf("error restoring pod with allocation ID %s: %w", msg.allocationID, err)
	}

	return resp, nil
}

func (j *jobsService) recreateGatewayProxyResources(
	services []k8sV1.Service,
	tcpRoutes []alphaGatewayTyped.TCPRoute,
	gatewayPorts []int,
) ([]gatewayProxyResource, error) {
	if j.internalTaskGWConfig == nil {
		return nil, nil
	}

	var resources []gatewayProxyResource
	for _, port := range gatewayPorts {
		var tcpRoute *alphaGatewayTyped.TCPRoute
		for _, t := range tcpRoutes {
			if len(t.Spec.ParentRefs) > 0 &&
				t.Spec.ParentRefs[0].Port != nil &&
				int(*t.Spec.ParentRefs[0].Port) == port {
				tcpRoute = &t
				break
			}
		}
		if tcpRoute == nil {
			return nil, fmt.Errorf("couldn't find tcpRoute for port %d", port)
		}

		var service *k8sV1.Service
		for _, s := range services {
			if s.Name == tcpRoute.Name {
				service = &s
				break
			}
		}
		if service == nil {
			return nil, fmt.Errorf("couldn't find service matching %s", tcpRoute.Name)
		}
		if len(service.Spec.Ports) != 1 {
			return nil, fmt.Errorf("expected service to have one port got %d", len(service.Spec.Ports))
		}
		resources = append(resources, gatewayProxyResource{
			serviceSpec:     service,
			tcpRouteSpec:    tcpRoute,
			gatewayListener: createListenerForPod(port),
		})
	}

	return resources, nil
}

func (j *jobsService) recreateJobHandler(
	name string,
	req *sproto.AllocateRequest,
	allocationID model.AllocationID,
	resourcePool string,
	job *batchV1.Job,
	slots int,
	numPods int,
	gatewayProxyResources []gatewayProxyResource,
	logContext logger.Context,
) (reattachJobResponse, error) {
	startMsg := startJob{
		req:          req,
		allocationID: allocationID,
		spec: tasks.TaskSpec{
			// This gets used in reattach to find the job by label its determinedLabel.
			AllocationID: string(allocationID),
			ContainerID:  req.AllocationID.String(), // ContainerID is non-sense, make a better abstraction.
		},
		slots:        slots,
		numPods:      numPods,
		resourcePool: resourcePool,
		logContext:   logContext,
	}

	newJobHandler := newJob(
		name,
		startMsg,
		startMsg.spec.ClusterID,
		j.clientSet,
		job.Namespace,
		j.detMasterIP,
		j.detMasterPort,
		j.detMasterScheme,
		j.masterTLSConfig,
		j.podInterfaces[job.Namespace],
		j.configMapInterfaces[job.Namespace],
		j.resourceRequestQueue,
		j.slotType,
		j.slotResourceRequests,
		j.scheduler,
		j.internalTaskGWConfig,
		j.gatewayService,
	)

	newJobHandler.restore = true
	newJobHandler.jobName = job.Name
	newJobHandler.configMapName = job.Name

	newJobHandler.gatewayProxyResources = gatewayProxyResources

	err := newJobHandler.startPodLogStreamers()
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

func (j *jobsService) deleteKubernetesResources(
	jobs []batchV1.Job,
	configMaps []k8sV1.ConfigMap,
	services []k8sV1.Service,
	tcpRoutes []alphaGatewayTyped.TCPRoute,
	gatewayPortsToFree []int,
) {
	for _, job := range jobs {
		j.resourceRequestQueue.deleteKubernetesResources(deleteKubernetesResources{
			namespace: job.Namespace,
			jobName:   job.Name,
		})
	}

	for _, configMap := range configMaps {
		j.resourceRequestQueue.deleteKubernetesResources(deleteKubernetesResources{
			namespace:     configMap.Namespace,
			configMapName: configMap.Name,
		})
	}

	for _, s := range services {
		j.resourceRequestQueue.deleteKubernetesResources(deleteKubernetesResources{
			namespace:    s.Namespace,
			serviceNames: []string{s.Name},
		})
	}

	for _, r := range tcpRoutes {
		j.resourceRequestQueue.deleteKubernetesResources(deleteKubernetesResources{
			namespace:     r.Namespace,
			tcpRouteNames: []string{r.Name},
		})
	}

	if len(gatewayPortsToFree) > 0 && j.internalTaskGWConfig != nil {
		j.resourceRequestQueue.deleteKubernetesResources(deleteKubernetesResources{
			namespace:          j.internalTaskGWConfig.GatewayNamespace,
			gatewayPortsToFree: gatewayPortsToFree,
		})
	}
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

func (j *jobsService) refreshJobState(allocationID model.AllocationID) error {
	if allocationID == "" {
		return fmt.Errorf("invalid call: allocationID missing")
	}

	jobs, err := j.listJobsInAllNamespaces(context.TODO(), metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", determinedLabel, allocationID),
	})
	if err != nil {
		return fmt.Errorf("error listing pods checking if they can be restored: %w", err)
	}

	ns, err := j.GetAllNamespacesForRM()
	if err != nil {
		return fmt.Errorf("failed to get namespaces for resource manager: %w", err)
	}

	for _, job := range jobs {
		if !slices.Contains(ns, job.Namespace) {
			continue
		}
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
		return fmt.Errorf("error listing pods checking if they can be restored: %w", err)
	}
	ns, err := j.GetAllNamespacesForRM()
	if err != nil {
		return fmt.Errorf("failed to get namespaces for resource manager: %w", err)
	}

	for _, pod := range pods {
		if !slices.Contains(ns, pod.Namespace) {
			continue
		}
		j.podStatusCallback(&pod)
	}
	return nil
}

func (j *jobsService) GetAgents() (*apiv1.GetAgentsResponse, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.getAgents()
}

func (j *jobsService) GetAgent(msg *apiv1.GetAgentRequest) *apiv1.GetAgentResponse {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.getAgent(msg.AgentId)
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

func (j *jobsService) GetSlots(msg *apiv1.GetSlotsRequest) *apiv1.GetSlotsResponse {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.getSlots(msg.AgentId)
}

func (j *jobsService) GetSlot(msg *apiv1.GetSlotRequest) *apiv1.GetSlotResponse {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.getSlot(msg.AgentId, msg.SlotId)
}

func (j *jobsService) HealthStatus(ctx context.Context) model.HealthStatus {
	if len(j.podInterfaces) == 0 {
		logrus.Error("expected podInterface to be non empty")
		return model.Unhealthy
	}

	_, err := j.podInterfaces[""].List(ctx, metaV1.ListOptions{Limit: 1})
	if k8error.IsForbidden(err) {
		return j.healthStatusFallback(ctx)
	} else if err != nil {
		return model.Unhealthy
	}
	return model.Healthy
}

func (j *jobsService) healthStatusFallback(ctx context.Context) model.HealthStatus {
	var g errgroup.Group
	for n, podInterface := range j.podInterfaces {
		if len(n) == 0 { // TODO: We store a non-namespaced client with key "".
			continue
		}
		g.Go(func() error {
			_, err := podInterface.List(ctx, metaV1.ListOptions{Limit: 1})
			if err != nil {
				return err
			}
			return nil
		})
	}
	err := g.Wait()
	if err != nil {
		return model.Unhealthy
	}
	return model.Healthy
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

func (j *jobsService) startEventListeners(namespace string, hasJSLock bool) error {
	callback := func(event watch.Event) {
		j.mu.Lock()
		defer j.mu.Unlock()
		j.newEventCallback(event)
	}
	if hasJSLock {
		callback = func(event watch.Event) {
			j.newEventCallback(event)
		}
	}

	l, err := newEventInformer(
		context.TODO(),
		j.clientSet.CoreV1().Events(namespace),
		namespace,
		callback,
	)
	if err != nil {
		return err
	}
	go l.run(context.TODO())

	return nil
}

func (j *jobsService) startPreemptionListeners(namespace string, hasJSLock bool) error {
	callback := func(event watch.Event) {
		j.mu.Lock()
		defer j.mu.Unlock()
		j.preemptionCallback(event)
	}
	if hasJSLock {
		callback = func(event watch.Event) {
			j.preemptionCallback(event)
		}
	}
	l, err := newPodInformer(
		context.TODO(),
		determinedPreemptionLabel,
		"preemption",
		namespace,
		j.clientSet.CoreV1().Pods(namespace),
		callback,
	)
	if err != nil {
		return err
	}
	go l.run(context.TODO())
	return nil
}

func (j *jobsService) startResourceRequestQueue() {
	failures := make(chan resourcesRequestFailure, 16)
	j.resourceRequestQueue, j.requestQueueWorkers = startRequestQueue(
		j.jobInterfaces,
		j.podInterfaces,
		j.configMapInterfaces,
		j.serviceInterfaces,
		j.gatewayService,
		j.tcpRouteInterfaces,
		failures)
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

	j.updatePodSchedulingState(jobName, *pod)
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

// updatePodSchedulingState stores the scheduling state of a pod based on its state.
func (j *jobsService) updatePodSchedulingState(jobName string, pod k8sV1.Pod) {
	states, ok := j.jobNameToPodNameToSchedulingState[jobName]
	if !ok {
		states = make(map[string]sproto.SchedulingState)
	}

	// The field pod.Spec.NodeName is a request to be scheduled onto a node but it is not guaranteed.
	states[pod.Name] = sproto.SchedulingStateQueued
	if podScheduled(pod) {
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
	for _, pod := range pods {
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

type computeUsageSummary struct {
	numAgentsUsed  int
	slotsAvailable int
}

func (j *jobsService) summarizeComputeUsage(poolName string) (*computeUsageSummary, error) {
	summary, err := j.summarize()
	if err != nil {
		return nil, err
	}

	slots := 0
	if len(poolName) > 0 {
		slots = numSlots(summary[poolName].Slots)
	} else {
		for _, pool := range summary {
			slots += numSlots(pool.Slots)
		}
	}
	return &computeUsageSummary{numAgentsUsed: len(summary), slotsAvailable: slots}, nil
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

func (j *jobsService) changePriority(id model.AllocationID) {
	ref, err := j.verifyJobAndGetRef(id)
	if err != nil {
		j.syslog.WithError(err).Debug("changing allocation priority")
		return
	}
	ref.changePriority()
}

func (j *jobsService) killJob(id model.AllocationID) {
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
		return fmt.Errorf("unknown job handler being deleted %s", jobHandler.jobName)
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

func (j *jobsService) getSlots(agentID string) *apiv1.GetSlotsResponse {
	agentResp := j.getAgent(agentID)
	if agentResp == nil {
		j.syslog.Warnf("no agent with id %s", agentID)
		return nil
	}
	return &apiv1.GetSlotsResponse{Slots: maps.Values(agentResp.Agent.Slots)}
}

func (j *jobsService) getSlot(agentID string, slotID string) *apiv1.GetSlotResponse {
	agentResp := j.getAgent(agentID)
	if agentResp == nil {
		j.syslog.Warnf("no agent with id %s", agentID)
		return nil
	}
	slots := agentResp.Agent.Slots
	slot, ok := slots[slotID]
	if !ok {
		// Try converting an index input to a slot and see if that exists (1 to 001).
		tryIndex, err := strconv.Atoi(slotID)
		s, ok := slots[model.SortableSlotIndex(tryIndex)]
		if err != nil || !ok {
			j.syslog.Warnf("no slot with id %s", slotID)
			return nil
		}
		slot = s
	}
	return &apiv1.GetSlotResponse{Slot: slot}
}

const getAgentsCacheDuration = 15 * time.Second

func (j *jobsService) getAgents() (*apiv1.GetAgentsResponse, error) {
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

	// Ensure cached response is not inadvertently modified.
	return rm.CopyGetAgentsResponse(j.getAgentsCache)
}

func (j *jobsService) getAgent(agentID string) *apiv1.GetAgentResponse {
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

const summarizeCacheDuration = 5 * time.Second

// summarize describes pods' available resources. When there's exactly one resource pool, it uses
// the whole cluster's info. Otherwise, it matches nodes to resource pools using node selectors, affinities,
// taints and tolerations to derive that info. This may be cached, so don't use this for decisions
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
		Key:      resourceTypeNvidia,
		Value:    "present",
		Operator: k8sV1.TolerationOpEqual,
	}}
	cpuTolerations, gpuTolerations := extractTolerations(j.baseContainerDefaults)
	poolsToNodes := make(map[string][]*k8sV1.Node)
	nodesToPools := make(map[string][]string)

	for _, node := range j.currentNodes {
		_, slotType := extractSlotInfo(nodeSummaries[node.Name])

		for poolName, tcd := range poolTaskContainerDefaults {
			var poolTolerations []k8sV1.Toleration
			var selectors, affinities *k8sV1.NodeSelector

			// If they're using the default RP config, use the default tolerations.
			// Don't check for node selectors or affinities here because the pod spec
			// isn't defined.
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
					selectors, affinities = extractNodeSelectors(tcd.GPUPodSpec)
				} else if tcd.CPUPodSpec != nil {
					//nolint:gocritic
					poolTolerations = append(tcd.CPUPodSpec.Spec.Tolerations, cpuTolerations...)
					selectors, affinities = extractNodeSelectors(tcd.CPUPodSpec)
				}
			}

			// add default toleration so that autoscaling nodes will still be counted.
			poolTolerations = append(poolTolerations, k8sV1.Toleration{
				Key:               "DeletionCandidateOfClusterAutoscaler",
				Operator:          "Exists",
				Effect:            "PreferNoSchedule",
				TolerationSeconds: nil,
			})

			// If all of a node's taints are tolerated by a pool & a node is a "match" to the pool's
			// node affinities and node selectors, that node belongs to the pool.
			if allTaintsTolerated(node.Spec.Taints, poolTolerations) &&
				j.podsCanBeScheduledOnNode(selectors, node) && j.podsCanBeScheduledOnNode(affinities, node) {
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
	summaries := make(map[string]model.AgentSummary)
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
			resources := node.Status.Allocatable[resourceTypeNvidia]
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
	for _, p := range allPods {
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
				reqs += c.Resources.Requests.Name(resourceTypeNvidia, resource.DecimalSI).Value()
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
	counts := make(map[string]int)
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
) ([]batchV1.Job, error) {
	allJobs, err := j.jobInterfaces[""].List(ctx, opts)
	if k8error.IsForbidden(err) {
		return j.listJobsInAllNamespacesFallback(ctx, opts)
	} else if err != nil {
		logrus.WithError(err).WithField("function", "listJobsInAllNamespaces").Error("error listing jobs in all namespace")
		return nil, err
	}

	namespaces := set.FromKeys(j.jobInterfaces)
	var jobsWeCareAbout []batchV1.Job
	for _, j := range allJobs.Items {
		if namespaces.Contains(j.Namespace) {
			jobsWeCareAbout = append(jobsWeCareAbout, j)
		}
	}
	return jobsWeCareAbout, nil
}

func (j *jobsService) listJobsInAllNamespacesFallback(
	ctx context.Context,
	opts metaV1.ListOptions,
) ([]batchV1.Job, error) {
	var g errgroup.Group
	var res []batchV1.Job
	var resLock sync.Mutex
	for n, i := range j.jobInterfaces {
		g.Go(func() error {
			pods, err := i.List(ctx, opts)
			if err != nil {
				return fmt.Errorf("error listing pods for namespace %s: %w", n, err)
			}
			resLock.Lock()
			res = append(res, pods.Items...)
			resLock.Unlock()
			return nil
		})
	}
	err := g.Wait()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (j *jobsService) listPodsInAllNamespaces(
	ctx context.Context, opts metaV1.ListOptions,
) ([]k8sV1.Pod, error) {
	allPods, err := j.podInterfaces[""].List(ctx, opts)
	if k8error.IsForbidden(err) {
		return j.listPodsInAllNamespacesFallback(ctx, opts)
	} else if err != nil {
		return nil, err
	}

	namespaces := set.FromKeys(j.podInterfaces)
	var podsWeWant []k8sV1.Pod
	for _, pod := range allPods.Items {
		if namespaces.Contains(pod.Namespace) {
			podsWeWant = append(podsWeWant, pod)
		}
	}
	return podsWeWant, nil
}

func (j *jobsService) listPodsInAllNamespacesFallback(
	ctx context.Context,
	opts metaV1.ListOptions,
) ([]k8sV1.Pod, error) {
	var g errgroup.Group
	var res []k8sV1.Pod
	var resLock sync.Mutex
	for n, podInterface := range j.podInterfaces {
		if len(n) == 0 {
			continue
		}
		g.Go(func() error {
			pods, err := podInterface.List(ctx, opts)
			if err != nil {
				return fmt.Errorf("error listing pods for namespace %s: %w", n, err)
			}
			resLock.Lock()
			res = append(res, pods.Items...)
			resLock.Unlock()
			return nil
		})
	}
	err := g.Wait()
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (j *jobsService) listConfigMapsInAllNamespaces(
	ctx context.Context, opts metaV1.ListOptions,
) ([]k8sV1.ConfigMap, error) {
	var res []k8sV1.ConfigMap
	for n, i := range j.configMapInterfaces {
		cms, err := i.List(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("error listing config maps for namespace %s: %w", n, err)
		}
		res = append(res, cms.Items...)
	}

	return res, nil
}

func (j *jobsService) listServicesInAllNamespaces(
	ctx context.Context, opts metaV1.ListOptions,
) ([]k8sV1.Service, error) {
	var res []k8sV1.Service
	for n, i := range j.serviceInterfaces {
		services, err := i.List(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("listing services for namespace %s: %w", n, err)
		}
		res = append(res, services.Items...)
	}

	return res, nil
}

func (j *jobsService) listTCPRoutesInAllNamespaces(
	ctx context.Context, opts metaV1.ListOptions,
) ([]alphaGatewayTyped.TCPRoute, error) {
	var res []alphaGatewayTyped.TCPRoute
	for n, i := range j.tcpRouteInterfaces {
		routes, err := i.List(ctx, opts)
		if err != nil {
			return nil, fmt.Errorf("listing TCPRoutes for namespace %s: %w", n, err)
		}
		res = append(res, routes.Items...)
	}

	return res, nil
}

func (j *jobsService) verifyNamespaceExists(namespace string, hasJSLock bool) error {
	_, err := j.clientSet.CoreV1().Namespaces().Get(context.Background(), namespace,
		metaV1.GetOptions{})
	if err != nil {
		return fmt.Errorf("error finding namespace %s: %w", namespace, err)
	}

	j.podInterfaces[namespace] = j.clientSet.CoreV1().Pods(namespace)
	j.configMapInterfaces[namespace] = j.clientSet.CoreV1().ConfigMaps(namespace)
	j.jobInterfaces[namespace] = j.clientSet.BatchV1().Jobs(namespace)

	for _, worker := range j.requestQueueWorkers {
		worker.podInterface = j.podInterfaces
		worker.configMapInterfaces = j.configMapInterfaces
		worker.jobInterface = j.jobInterfaces
	}

	err = j.syncNamespaces([]string{namespace}, true)
	if err != nil {
		return err
	}

	return nil
}

func (j *jobsService) createNamespace(namespaceName string, hasJSLock bool) error {
	err := j.createNamespaceHelper(namespaceName)
	if err != nil {
		return err
	}

	err = j.syncNamespaces([]string{namespaceName}, true)
	if err != nil {
		return err
	}

	return nil
}

func (j *jobsService) createNamespaceHelper(namespaceName string) error {
	_, err := j.clientSet.CoreV1().Namespaces().Create(
		context.TODO(),
		&k8sV1.Namespace{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
			ObjectMeta: metaV1.ObjectMeta{
				Name:   namespaceName,
				Labels: map[string]string{determinedLabel: namespaceName},
			},
		},
		metaV1.CreateOptions{},
	)
	if err != nil {
		if !k8error.IsAlreadyExists(err) {
			return fmt.Errorf("error creating namespace %s: %w", namespaceName, err)
		}
		return nil
	}

	j.podInterfaces[namespaceName] = j.clientSet.CoreV1().Pods(namespaceName)
	j.configMapInterfaces[namespaceName] = j.clientSet.CoreV1().ConfigMaps(namespaceName)
	j.jobInterfaces[namespaceName] = j.clientSet.BatchV1().Jobs(namespaceName)

	for _, worker := range j.requestQueueWorkers {
		worker.podInterface = j.podInterfaces
		worker.configMapInterfaces = j.configMapInterfaces
		worker.jobInterface = j.jobInterfaces
	}

	return nil
}

func (j *jobsService) deleteNamespace(namespaceName string) error {
	err := j.clientSet.CoreV1().Namespaces().Delete(context.TODO(), namespaceName,
		metaV1.DeleteOptions{})
	if err != nil && !k8error.IsNotFound(err) {
		return err
	}
	return nil
}

func (j *jobsService) removeEmptyNamespace(namespaceName string, clusterName string) error {
	count, err := workspace.GetNumWorkspacesUsingNamespaceInCluster(context.Background(), clusterName, namespaceName)
	if err != nil {
		return err
	}
	if count == 0 {
		delete(j.podInterfaces, namespaceName)
		delete(j.configMapInterfaces, namespaceName)
		delete(j.jobInterfaces, namespaceName)

		for _, worker := range j.requestQueueWorkers {
			worker.podInterface = j.podInterfaces
			worker.configMapInterfaces = j.configMapInterfaces
			worker.jobInterface = j.jobInterfaces
		}
	}
	return nil
}

func (j *jobsService) setResourceQuota(quota int, namespace string) error {
	k8sDeterminedLabel := map[string]string{determinedLabel: namespace}

	k8sNamespace, err := j.clientSet.CoreV1().Namespaces().Get(context.TODO(), namespace,
		metaV1.GetOptions{
			TypeMeta: metaV1.TypeMeta{
				Kind:       "Namespace",
				APIVersion: "v1",
			},
		})
	if err != nil {
		return fmt.Errorf("error finding namespace %s: %w", namespace, err)
	}

	if _, ok := k8sNamespace.Labels[determinedLabel]; !ok {
		return fmt.Errorf("cannot set quota on namespace %s. Namespace needs determined label",
			namespace)
	}

	quotaName := namespace + "-quota"

	// We want to patch the smallest quota if it is a determiend quota regardless of whether it's
	// larger or smaller. If the quota does not correspond to the auto-generated namespace and is
	// less than the current quota (the first non-Determined quota, we error out saying that they
	// should remove that quota from Kubernetes and then try to set a Determined quota.
	k8sQuotas, err := j.clientSet.CoreV1().ResourceQuotas(namespace).List(context.TODO(),
		metaV1.ListOptions{})
	if err != nil {
		return fmt.Errorf("error fetching resource quotas for namespace %s: %w", namespace, err)
	}
	quotas := k8sQuotas.Items
	currentQuota := float64(quota)
	var detQuota *k8sV1.ResourceQuota
	for _, q := range quotas {
		qResources := q.Spec.Hard
		for name, quantity := range qResources {
			if name == "requests."+ResourceTypeNvidia {
				tmpQuota := quantity.AsApproximateFloat64()
				q.Spec.Hard[name] = *resource.NewQuantity(int64(quota), resource.DecimalSI)
				if q.Name == quotaName {
					qVal := q
					detQuota = &qVal
				} else if tmpQuota < currentQuota {
					lowerQuotaName := q.Name
					return fmt.Errorf("cannot set quota %d on namespace %s because this"+
						" namespace contains resource quota %s with GPU request limit %d, which is"+
						" lower than the request limit you wish to set on this namespace. Please"+
						" remove this quota in Kubernetes before trying to raise the GPU request"+
						" limit on the namespace",
						quota, namespace, lowerQuotaName, int(tmpQuota))
				}
			}
		}
	}

	if detQuota != nil {
		detQuotaToByteArray, err := json.Marshal(detQuota)
		if err != nil {
			return fmt.Errorf("error marshaling quota %s: %w", detQuota.Name, err)
		}
		_, err = j.clientSet.CoreV1().ResourceQuotas(namespace).Patch(context.TODO(),
			quotaName,
			types.MergePatchType,
			detQuotaToByteArray,
			metaV1.PatchOptions{})
		if err != nil {
			return fmt.Errorf("error applying patch to resource quota %s: %w", quotaName, err)
		}
	} else {
		// The given namespace does not any attached determined quotas.
		if currentQuota < float64(quota) {
			return fmt.Errorf("cannot set quota because there already exists a quota in "+
				"namespace %s of limit %d", namespace, int(currentQuota))
		}
		_, err = j.clientSet.CoreV1().ResourceQuotas(namespace).Create(context.TODO(),
			&k8sV1.ResourceQuota{
				TypeMeta: metaV1.TypeMeta{
					Kind:       "ResourceQuota",
					APIVersion: "v1",
				},
				ObjectMeta: metaV1.ObjectMeta{Labels: k8sDeterminedLabel, Name: quotaName},
				Spec: k8sV1.ResourceQuotaSpec{
					Hard: k8sV1.ResourceList{
						k8sV1.ResourceName("requests." + ResourceTypeNvidia): *resource.
							NewQuantity(int64(quota), resource.DecimalSI),
					},
				},
			},
			metaV1.CreateOptions{},
		)
		if err != nil {
			return fmt.Errorf("error creating resource quota %s for namespace %s: %w",
				quotaName, namespace, err)
		}
	}

	return nil
}

func (j *jobsService) getNamespaceResourceQuota(namespaceName string) (*float64, error) {
	k8sQuotas, err := j.clientSet.CoreV1().ResourceQuotas(namespaceName).List(context.TODO(), metaV1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error finding resource quotas for the namespace %s: %w", namespaceName, err)
	}

	quotas := k8sQuotas.Items
	minQuota := math.Inf(1)
	for _, q := range quotas {
		qResources := q.Spec.Hard
		for name, quantity := range qResources {
			if name == "requests."+resourceTypeNvidia {
				minQuota = math.Min(minQuota, quantity.AsApproximateFloat64())
			}
		}
	}

	if minQuota != math.Inf(1) {
		return &minQuota, nil
	}

	return nil, nil
}

func (j *jobsService) getInitialNamespace() string {
	releaseNamespace := os.Getenv(ReleaseNamespaceEnvVar)
	if len(releaseNamespace) > 0 {
		return releaseNamespace
	}
	return j.namespace
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

// Check that pods belong to this resource pool (from poolTCD) can be scheduled on a given node.
func (j *jobsService) podsCanBeScheduledOnNode(selector *k8sV1.NodeSelector, node *k8sV1.Node) bool {
	// In case of no defined affinities/node selectors, the pod can default to schedule on the given node.
	if selector == nil || len(selector.NodeSelectorTerms) == 0 {
		return true
	}

	ns, err := nodeaffinity.NewNodeSelector(selector)
	if err != nil {
		j.syslog.WithError(err)
		return false
	}
	return ns.Match(node)
}

// Gets the node affinity/selector from the resource pool.
func extractNodeSelectors(pod *k8sV1.Pod) (selectors, affinities *k8sV1.NodeSelector) {
	if pod == nil {
		return nil, nil
	}

	// First check for node affinities.
	if nodeAffinity := pod.Spec.Affinity; nodeAffinity != nil {
		affinities = nodeAffinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	}

	// Then add any node labels.
	if nodeSelector := pod.Spec.NodeSelector; nodeSelector != nil {
		selectors = &k8sV1.NodeSelector{}
		expr := []k8sV1.NodeSelectorRequirement{}
		for k, v := range nodeSelector {
			expr = append(expr, k8sV1.NodeSelectorRequirement{
				Key:      k,
				Operator: k8sV1.NodeSelectorOpIn,
				Values:   strings.Split(v, ","),
			})
		}
		selectors.NodeSelectorTerms = []k8sV1.NodeSelectorTerm{{MatchExpressions: expr}}
	}
	return selectors, affinities
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
