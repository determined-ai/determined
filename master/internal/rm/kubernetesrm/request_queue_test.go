package kubernetesrm

import (
	"context"
	"fmt"
	"testing"
	"time"

	petName "github.com/dustinkirkland/golang-petname"
	"github.com/sirupsen/logrus"
	"gotest.tools/assert"

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type mockPod struct {
	requestQueue *requestQueue
	name         string
	errorHandler errorCallbackFunc
	syslog       *logrus.Entry
}

func startMockPod(requestQueue *requestQueue, errorHandler *errorCallbackFunc) *mockPod {
	m := &mockPod{
		requestQueue: requestQueue,
		name:         petName.Generate(3, "-"),
	}
	if errorHandler == nil {
		m.errorHandler = m.defaultErrorHandler
	} else {
		m.errorHandler = *errorHandler
	}
	m.syslog = logrus.New().WithField("component", "kubernetesrm-mock-pod").WithField("name", m.name)
	m.create()
	return m
}

func (m *mockPod) defaultErrorHandler(e error) {
	switch e := e.(type) {
	case resourceCreationFailed:
		m.syslog.Errorf("defaultErrorHandler resource creation failed: %v", e)
	case resourceDeletionFailed:
		m.syslog.Errorf("defaultErrorHandler resource deletion failed: %v", e)
	case resourceCreationCancelled:
		m.syslog.Infof("defaultErrorHandler resource deletion failed: %v", e)
	default:
		panic(fmt.Sprintf("unexpected error %T", e))
	}
}

func (m *mockPod) create() {
	podSpec := k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{
		Name:      m.name,
		Namespace: "default",
	}}
	cmSpec := k8sV1.ConfigMap{ObjectMeta: metaV1.ObjectMeta{
		Name:      m.name,
		Namespace: "default",
	}}
	m.requestQueue.createKubernetesResources(m.errorHandler, &podSpec, &cmSpec)
}

func (m *mockPod) delete() {
	m.requestQueue.deleteKubernetesResources(m.errorHandler, "default", m.name, m.name)
}

func getNumberOfActivePods(podInterface typedV1.PodInterface) int {
	podList, err := podInterface.List(context.TODO(), metaV1.ListOptions{})
	if err != nil {
		panic(err)
	}

	return len(podList.Items)
}

func requestQueueIsProcessing(r *requestQueue) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.queue) > 0 &&
		len(r.blockedResourceDeletions) == 0 &&
		r.availableWorkers < numKubernetesWorkers
}

func waitForPendingRequestToFinish(k8RequestQueue *requestQueue) {
	time.Sleep(time.Second)

	// Wait for queue to finish all in flight requests.
	for requestQueueIsProcessing(k8RequestQueue) {
		time.Sleep(time.Millisecond * 100)
	}
}

func deleteAll(pods []*mockPod) {
	for _, p := range pods {
		p.delete()
	}
}

func TestRequestQueueCreatingManyPod(t *testing.T) {
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	numPods := 15
	for i := 0; i < numPods; i++ {
		startMockPod(k8sRequestQueue, nil)
	}

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), numPods)
}

func TestRequestQueueCreatingAndDeletingManyPod(t *testing.T) {
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	numPods := 15
	pods := make([]*mockPod, 0)
	for i := 0; i < numPods; i++ {
		pods = append(pods, startMockPod(k8sRequestQueue, nil))
	}
	deleteAll(pods)

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), 0)
}

func TestRequestQueueCreatingThenDeletingManyPods(t *testing.T) {
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	numPods := 15
	pods := make([]*mockPod, 0)
	for i := 0; i < numPods; i++ {
		pods = append(pods, startMockPod(k8sRequestQueue, nil))
	}

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), numPods)

	deleteAll(pods)

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), 0)
}

func TestRequestQueueCreatingAndDeletingManyPodWithDelay(t *testing.T) {
	podInterface := &mockPodInterface{
		pods:             make(map[string]*k8sV1.Pod),
		operationalDelay: time.Millisecond * 500,
	}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	numPods := 15
	pods := make([]*mockPod, 0)
	for i := 0; i < numPods; i++ {
		pods = append(pods, startMockPod(k8sRequestQueue, nil))
	}
	deleteAll(pods)

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, getNumberOfActivePods(podInterface), 0)
}

func TestRequestQueueCreationCancelled(t *testing.T) {
	podInterface := &mockPodInterface{
		pods:             make(map[string]*k8sV1.Pod),
		operationalDelay: time.Millisecond * 500,
	}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	for i := 0; i < numKubernetesWorkers; i++ {
		startMockPod(k8sRequestQueue, nil)
	}

	createCancelled := false
	errorHandler := (errorCallbackFunc)(func(e error) {
		switch e := e.(type) {
		case resourceCreationCancelled:
			createCancelled = true
		default:
			panic(fmt.Sprintf("unexpected error %T", e))
		}
	})
	pod := startMockPod(k8sRequestQueue, &errorHandler)
	assert.Equal(t, createCancelled, false)
	pod.delete()
	time.Sleep(time.Second / 10)
	assert.Equal(t, createCancelled, true)
}

func TestRequestQueueCreationFailed(t *testing.T) {
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	createFailed := false
	errorHandler := (errorCallbackFunc)(func(e error) {
		switch e := e.(type) {
		case resourceCreationFailed:
			createFailed = true
		default:
			panic(fmt.Sprintf("unexpected error %T", e))
		}
	})
	pod := startMockPod(k8sRequestQueue, &errorHandler)
	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, createFailed, false)

	pod.create()
	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, createFailed, true)
}

func TestRequestQueueDeletionFailed(t *testing.T) {
	podInterface := &mockPodInterface{pods: make(map[string]*k8sV1.Pod)}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	deleteFailed := false
	errorHandler := (errorCallbackFunc)(func(e error) {
		switch e := e.(type) {
		case resourceDeletionFailed:
			deleteFailed = true
		default:
			panic(fmt.Sprintf("unexpected error %T", e))
		}
	})
	pod := startMockPod(k8sRequestQueue, &errorHandler)
	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, deleteFailed, false)

	pod.delete()
	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, deleteFailed, false)

	pod.delete()
	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, deleteFailed, true)
}
