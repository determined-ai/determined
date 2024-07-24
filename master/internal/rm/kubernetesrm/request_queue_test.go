package kubernetesrm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	petName "github.com/dustinkirkland/golang-petname"
	"github.com/sirupsen/logrus"
	"gotest.tools/assert"

	batchV1 "k8s.io/api/batch/v1"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedBatchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
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
	m.requestQueue.createKubernetesResources(&jobSpec, &cmSpec, nil)
}

func (m *mockJob) delete() {
	m.requestQueue.deleteKubernetesResources(deleteKubernetesResources{
		namespace:     "default",
		jobName:       m.name,
		configMapName: m.name,
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
	k8sRequestQueue, _ := startRequestQueue(
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
	k8sRequestQueue, _ := startRequestQueue(
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
	k8sRequestQueue, _ := startRequestQueue(
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
	k8sRequestQueue, _ := startRequestQueue(
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
	k8sRequestQueue, _ := startRequestQueue(
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
	k8sRequestQueue, _ := startRequestQueue(
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
	k8sRequestQueue, _ := startRequestQueue(
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
