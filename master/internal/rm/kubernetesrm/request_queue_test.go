package kubernetesrm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	petName "github.com/dustinkirkland/golang-petname"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"gotest.tools/assert"

	batchV1 "k8s.io/api/batch/v1"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedBatchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
	gatewayTyped "sigs.k8s.io/gateway-api/apis/v1"
	alphaGatewayTyped "sigs.k8s.io/gateway-api/apis/v1alpha2"
	alphaGateway "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/typed/apis/v1alpha2"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

type mockJob struct {
	requestQueue *requestQueue
	name         string
	syslog       *logrus.Entry
}

func startMockJob(requestQueue *requestQueue) *mockJob {
	m := &mockJob{
		requestQueue: requestQueue,
		name:         petName.Generate(3, "-"),
	}
	m.syslog = logrus.WithField("component", "kubernetesrm-mock-pod").WithField("name", m.name)
	m.create()
	return m
}

func runDefaultErrorHandler(ctx context.Context, failures <-chan resourcesRequestFailure) {
	for {
		select {
		case failure := <-failures:
			switch e := failure.(type) {
			case resourceCreationFailed:
				logrus.Errorf("defaultErrorHandler resource creation failed: %v", e)
			case resourceDeletionFailed:
				logrus.Errorf("defaultErrorHandler resource deletion failed: %v", e)
			case resourceCreationCancelled:
				logrus.Infof("defaultErrorHandler resource deletion failed: %v", e)
			default:
				panic(fmt.Sprintf("unexpected error %T", e))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (m *mockJob) create() {
	jobSpec := batchV1.Job{ObjectMeta: metaV1.ObjectMeta{
		Name:      m.name,
		Namespace: "default",
	}}
	cmSpec := k8sV1.ConfigMap{ObjectMeta: metaV1.ObjectMeta{
		Name:      m.name,
		Namespace: "default",
	}}
	m.requestQueue.createKubernetesResources(&podSpec, &cmSpec, nil)
}

func (m *mockPod) delete() {
	m.requestQueue.deleteKubernetesResources(deleteKubernetesResources{
		namespace:     "default",
		podName:       &m.name,
		configMapName: &m.name,
	})
}

func getNumberOfActiveJobs(jobInterface typedBatchV1.JobInterface) int {
	jobList, err := jobInterface.List(context.TODO(), metaV1.ListOptions{})
	if err != nil {
		panic(err)
	}

	return len(jobList.Items)
}

func requestQueueIsDone(r *requestQueue) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.queue) == 0 &&
		len(r.blockedResourceDeletions) == 0 &&
		len(r.pendingResourceCreations) == 0 &&
		len(r.creationInProgress) == 0
}

func waitForPendingRequestToFinish(k8RequestQueue *requestQueue) {
	// Wait for queue to finish all in flight requests.
	for !requestQueueIsDone(k8RequestQueue) {
		time.Sleep(time.Millisecond * 100)
	}
	time.Sleep(time.Second)
}

func deleteAll(pods []*mockJob) {
	for _, p := range pods {
		p.delete()
	}
}

func TestRequestQueueCreatingManyPod(t *testing.T) {
	jobInterface := &mockJobInterface{jobs: make(map[string]*batchV1.Job)}
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	failures := make(chan resourcesRequestFailure, 64)
	k8sRequestQueue := startRequestQueue(
		map[string]typedBatchV1.JobInterface{"default": jobInterface},
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		nil, nil, nil, failures,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runDefaultErrorHandler(ctx, failures)

	numPods := 15
	for i := 0; i < numPods; i++ {
		startMockJob(k8sRequestQueue)
	}

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActiveJobs(jobInterface), numPods)
}

func TestRequestQueueCreatingAndDeletingManyPod(t *testing.T) {
	jobInterface := &mockJobInterface{jobs: make(map[string]*batchV1.Job)}
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	failures := make(chan resourcesRequestFailure, 64)
	k8sRequestQueue := startRequestQueue(
		map[string]typedBatchV1.JobInterface{"default": jobInterface},
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		nil, nil, nil, failures,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runDefaultErrorHandler(ctx, failures)

	numPods := 15
	pods := make([]*mockJob, 0)
	for i := 0; i < numPods; i++ {
		pods = append(pods, startMockJob(k8sRequestQueue))
	}
	deleteAll(pods)

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActiveJobs(jobInterface), 0)
}

func TestRequestQueueCreatingThenDeletingManyPods(t *testing.T) {
	jobInterface := &mockJobInterface{jobs: make(map[string]*batchV1.Job)}
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	failures := make(chan resourcesRequestFailure, 64)
	k8sRequestQueue := startRequestQueue(
		map[string]typedBatchV1.JobInterface{"default": jobInterface},
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		nil, nil, nil, failures,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runDefaultErrorHandler(ctx, failures)

	numPods := 15
	pods := make([]*mockJob, 0)
	for i := 0; i < numPods; i++ {
		pods = append(pods, startMockJob(k8sRequestQueue))
	}

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActiveJobs(jobInterface), numPods)

	deleteAll(pods)

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActiveJobs(jobInterface), 0)
}

func TestRequestQueueCreatingAndDeletingManyPodWithDelay(t *testing.T) {
	jobInterface := &mockJobInterface{
		jobs:             make(map[string]*batchV1.Job),
		operationalDelay: time.Millisecond * 500,
	}
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod), operationalDelay: time.Millisecond * 500}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	failures := make(chan resourcesRequestFailure, 64)
	k8sRequestQueue := startRequestQueue(
		map[string]typedBatchV1.JobInterface{"default": jobInterface},
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		nil, nil, nil, failures,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runDefaultErrorHandler(ctx, failures)

	numPods := 15
	pods := make([]*mockJob, 0)
	for i := 0; i < numPods; i++ {
		pods = append(pods, startMockJob(k8sRequestQueue))
	}
	deleteAll(pods)

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActiveJobs(jobInterface), 0)
}

func TestRequestQueueCreationCancelled(t *testing.T) {
	jobInterface := &mockJobInterface{
		jobs:             make(map[string]*batchV1.Job),
		operationalDelay: time.Millisecond * 500,
	}
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod), operationalDelay: time.Millisecond * 500}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	failures := make(chan resourcesRequestFailure, 64)
	k8sRequestQueue := startRequestQueue(
		map[string]typedBatchV1.JobInterface{"default": jobInterface},
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		nil, nil, nil, failures,
	)

	for i := 0; i < numKubernetesWorkers; i++ {
		startMockJob(k8sRequestQueue)
	}
	time.Sleep(time.Millisecond * 100)

	var wg sync.WaitGroup
	wg.Add(1)
	createCancelled := false
	go func() {
		defer wg.Done()
		for failure := range failures {
			switch e := failure.(type) {
			case resourceCreationCancelled:
				createCancelled = true
				return
			default:
				panic(fmt.Sprintf("unexpected error %T", e))
			}
		}
	}()

	pod := startMockJob(k8sRequestQueue)
	assert.Equal(t, createCancelled, false)
	pod.delete()
	wg.Wait()
	assert.Equal(t, createCancelled, true)
}

func TestRequestQueueCreationFailed(t *testing.T) {
	jobInterface := &mockJobInterface{jobs: make(map[string]*batchV1.Job)}
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	failures := make(chan resourcesRequestFailure, 64)
	k8sRequestQueue := startRequestQueue(
		map[string]typedBatchV1.JobInterface{"default": jobInterface},
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		nil, nil, nil, failures,
	)

	var wg sync.WaitGroup
	wg.Add(1)
	createFailed := false
	go func() {
		defer wg.Done()
		for failure := range failures {
			switch e := failure.(type) {
			case resourceCreationFailed:
				createFailed = true
				return
			default:
				panic(fmt.Sprintf("unexpected error %T", e))
			}
		}
	}()

	pod := startMockJob(k8sRequestQueue)
	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, createFailed, false)

	pod.create()
	wg.Wait()
	assert.Equal(t, createFailed, true)
}

func TestRequestQueueDeletionFailed(t *testing.T) {
	jobInterface := &mockJobInterface{jobs: make(map[string]*batchV1.Job)}
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	failures := make(chan resourcesRequestFailure, 64)
	k8sRequestQueue := startRequestQueue(
		map[string]typedBatchV1.JobInterface{"default": jobInterface},
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
		nil, nil, nil, failures,
	)

	var wg sync.WaitGroup
	wg.Add(1)
	deleteFailed := false
	go func() {
		defer wg.Done()
		for failure := range failures {
			switch e := failure.(type) {
			case resourceDeletionFailed:
				deleteFailed = true
				return
			default:
				panic(fmt.Sprintf("unexpected error %T", e))
			}
		}
	}()

	pod := startMockJob(k8sRequestQueue)
	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, deleteFailed, false)

	pod.delete()
	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, deleteFailed, false)

	pod.delete()
	wg.Wait()
	assert.Equal(t, deleteFailed, true)
}

func TestReceiveCreateKubernetesResources(t *testing.T) {
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mocks.ConfigMapInterface{}
	serviceInterface := &mocks.ServiceInterface{}
	tcpInterface := &mocks.TCPRouteInterface{}
	gatewayInterface := &mocks.GatewayInterface{}

	w := &requestProcessingWorker{
		syslog:              logrus.New().WithField("test", "test"),
		podInterfaces:       map[string]typedV1.PodInterface{"": podInterface},
		configMapInterfaces: map[string]typedV1.ConfigMapInterface{"": configMapInterface},
		serviceInterfaces:   map[string]typedV1.ServiceInterface{"": serviceInterface},
		tcpRouteInterfaces:  map[string]alphaGateway.TCPRouteInterface{"": tcpInterface},
		gatewayService: &gatewayService{
			gatewayInterface: gatewayInterface,
			gatewayName:      "gatewayname",
		},
	}

	createReq := createKubernetesResources{
		podSpec:       &k8sV1.Pod{},
		configMapSpec: &k8sV1.ConfigMap{},
		gatewayProxyResources: []gatewayProxyResource{
			{
				serviceSpec:     &k8sV1.Service{},
				tcpRouteSpec:    &alphaGatewayTyped.TCPRoute{},
				gatewayListener: gatewayTyped.Listener{},
			},
		},
	}

	gateway := &gatewayTyped.Gateway{}
	expectedUpdatedGateway := &gatewayTyped.Gateway{
		Spec: gatewayTyped.GatewaySpec{
			Listeners: []gatewayTyped.Listener{
				{
					Port: 0,
				},
			},
		},
	}

	podInterface.On("Create", mock.Anything, createReq.podSpec, metaV1.CreateOptions{}).
		Return(createReq.podSpec, nil)
	configMapInterface.On("Create", mock.Anything, createReq.configMapSpec, metaV1.CreateOptions{}).
		Return(createReq.configMapSpec, nil)
	serviceInterface.On("Create", mock.Anything, createReq.gatewayProxyResources[0].serviceSpec,
		metaV1.CreateOptions{}).Return(createReq.gatewayProxyResources[0].serviceSpec, nil)
	tcpInterface.On("Create", mock.Anything, createReq.gatewayProxyResources[0].tcpRouteSpec,
		metaV1.CreateOptions{}).Return(createReq.gatewayProxyResources[0].tcpRouteSpec, nil)

	gatewayInterface.On("Get", mock.Anything, "gatewayname", metaV1.GetOptions{}).
		Return(gateway, nil)
	gatewayInterface.On("Update", mock.Anything, expectedUpdatedGateway, metaV1.UpdateOptions{}).
		Return(nil, nil)

	w.receiveCreateKubernetesResources(createReq)

	podInterface.AssertExpectations(t)
	configMapInterface.AssertExpectations(t)
	serviceInterface.AssertExpectations(t)
	tcpInterface.AssertExpectations(t)
	gatewayInterface.AssertExpectations(t)
}

func TestReceiveDeleteKubernetesResources(t *testing.T) {
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mocks.ConfigMapInterface{}
	serviceInterface := &mocks.ServiceInterface{}
	tcpInterface := &mocks.TCPRouteInterface{}
	gatewayInterface := &mocks.GatewayInterface{}

	w := &requestProcessingWorker{
		syslog:              logrus.New().WithField("test", "test"),
		podInterfaces:       map[string]typedV1.PodInterface{"": podInterface},
		configMapInterfaces: map[string]typedV1.ConfigMapInterface{"": configMapInterface},
		serviceInterfaces:   map[string]typedV1.ServiceInterface{"": serviceInterface},
		tcpRouteInterfaces:  map[string]alphaGateway.TCPRouteInterface{"": tcpInterface},
		gatewayService: &gatewayService{
			gatewayInterface: gatewayInterface,
			gatewayName:      "gatewayname",
		},
	}

	deleteReq := deleteKubernetesResources{
		podName:            ptrs.Ptr("podName"),
		configMapName:      ptrs.Ptr("configMapName"),
		serviceNames:       []string{"serviceName"},
		tcpRouteNames:      []string{"tcpRouteName"},
		gatewayPortsToFree: []int{1},
	}

	gateway := &gatewayTyped.Gateway{
		Spec: gatewayTyped.GatewaySpec{
			Listeners: []gatewayTyped.Listener{
				{
					Port: 1,
				},
			},
		},
	}
	expectedUpdatedGateway := &gatewayTyped.Gateway{}

	podInterface.On("Delete", mock.Anything, "podName", mock.Anything).Return(nil)
	configMapInterface.On("Delete", mock.Anything, "configMapName", mock.Anything).Return(nil)
	serviceInterface.On("Delete", mock.Anything, "serviceName", mock.Anything).Return(nil)
	tcpInterface.On("Delete", mock.Anything, "tcpRouteName", mock.Anything).Return(nil)

	gatewayInterface.On("Get", mock.Anything, "gatewayname", metaV1.GetOptions{}).
		Return(gateway, nil)
	gatewayInterface.On("Update", mock.Anything, expectedUpdatedGateway, metaV1.UpdateOptions{}).
		Return(nil, nil)

	w.receiveDeleteKubernetesResources(deleteReq)

	podInterface.AssertExpectations(t)
	configMapInterface.AssertExpectations(t)
	serviceInterface.AssertExpectations(t)
	tcpInterface.AssertExpectations(t)
	gatewayInterface.AssertExpectations(t)
}
