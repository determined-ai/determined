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

	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/internal/mocks"
)

type mockPod struct {
	requestQueue *requestQueue
	name         string
	errorHandler errorCallbackFunc
	syslog       *logrus.Entry
}

func startMockPod(
	requestQueue *requestQueue,
	name string,
	errorHandler *errorCallbackFunc,
) *mockPod {
	m := &mockPod{
		requestQueue: requestQueue,
		name:         name,
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

func deleteAll(pods []*mockPod) {
	for _, p := range pods {
		p.delete()
	}
}

func podList(m map[string]*k8sV1.Pod) *k8sV1.PodList {
	podList := &k8sV1.PodList{}
	for _, pod := range m {
		podList.Items = append(podList.Items, *pod)
	}
	return podList
}

func TestRequestQueueCreatingManyPod(t *testing.T) {
	pods := make(map[string]*k8sV1.Pod)
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	numPods := 15
	for i := 0; i < numPods; i++ {
		name := petName.Generate(3, "-")
		pod := &k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: "default"}}
		podInterface.On("Create", context.TODO(), pod,
			metaV1.CreateOptions{}).Return(&k8sV1.Pod{}, nil).Run(
			func(args mock.Arguments) { pods[name] = pod.DeepCopy() })
		startMockPod(k8sRequestQueue, name, nil)
	}

	waitForPendingRequestToFinish(k8sRequestQueue)
	podInterface.On("List", context.TODO(), metaV1.ListOptions{}).Return(podList(pods), nil)
	assert.Equal(t, len(podList(pods).Items), numPods)
}

func TestRequestQueueCreatingAndDeletingManyPod(t *testing.T) {
	pods := make(map[string]*k8sV1.Pod)
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	numPods := 1
	tmpPods := make([]*mockPod, 0)
	for i := 0; i < numPods; i++ {
		name := petName.Generate(3, "-")
		gracePeriod := int64(15)
		pod := &k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: "default"}}

		podInterface.On("Create", context.TODO(), pod, metaV1.CreateOptions{}).Return(
			&k8sV1.Pod{}, nil).Run(func(args mock.Arguments) { pods[name] = pod.DeepCopy() })
		podInterface.On("Delete", context.TODO(), name,
			metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod}).Return(nil).Run(
			func(args mock.Arguments) { delete(pods, name) })

		tmpPods = append(tmpPods, startMockPod(k8sRequestQueue, name, nil))
	}
	waitForPendingRequestToFinish(k8sRequestQueue)
	deleteAll(tmpPods)

	waitForPendingRequestToFinish(k8sRequestQueue)
	podInterface.On("List", context.TODO(), metaV1.ListOptions{}).Return(podList(pods), nil)
	assert.Equal(t, len(podList(pods).Items), 0)
}

func TestRequestQueueCreatingThenDeletingManyPods(t *testing.T) {
	pods := make(map[string]*k8sV1.Pod)
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	numPods := 15
	tmpPods := make([]*mockPod, 0)
	for i := 0; i < numPods; i++ {
		name := petName.Generate(3, "-")
		pod := &k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: "default"}}

		podInterface.On("Create", context.TODO(), pod, metaV1.CreateOptions{}).Return(
			&k8sV1.Pod{}, nil).Run(func(args mock.Arguments) { pods[name] = pod.DeepCopy() })

		gracePeriod := int64(15)
		podInterface.On("Delete", context.TODO(), name,
			metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod}).Return(nil).Run(
			func(args mock.Arguments) { delete(pods, name) })

		tmpPods = append(tmpPods, startMockPod(k8sRequestQueue, name, nil))
	}

	waitForPendingRequestToFinish(k8sRequestQueue)
	podInterface.On("List", context.TODO(), metaV1.ListOptions{}).Return(podList(pods), nil)
	assert.Equal(t, len(podList(pods).Items), numPods)

	deleteAll(tmpPods)

	waitForPendingRequestToFinish(k8sRequestQueue)
	podInterface.On("List", context.TODO(), metaV1.ListOptions{}).Return(podList(pods), nil)
	assert.Equal(t, len(podList(pods).Items), 0)
}

func TestRequestQueueCreatingAndDeletingManyPodWithDelay(t *testing.T) {
	pods := make(map[string]*k8sV1.Pod)
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	numPods := 15
	tmpPods := make([]*mockPod, 0)
	for i := 0; i < numPods; i++ {
		name := petName.Generate(3, "-")
		pod := &k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: "default"}}

		podInterface.On("Create", context.TODO(), pod, metaV1.CreateOptions{}).Return(
			&k8sV1.Pod{}, nil).Run(func(args mock.Arguments) { pods[name] = pod.DeepCopy() })

		gracePeriod := int64(15)
		podInterface.On("Delete", context.TODO(), name,
			metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod}).Return(nil).Run(
			func(args mock.Arguments) { delete(pods, name) })

		tmpPods = append(tmpPods, startMockPod(k8sRequestQueue, name, nil))
	}
	deleteAll(tmpPods)

	waitForPendingRequestToFinish(k8sRequestQueue)
	podInterface.On("List", context.TODO(), metaV1.ListOptions{}).Return(podList(pods), nil)
	assert.Equal(t, len(podList(pods).Items), 0)
}

func TestRequestQueueCreationCancelled(t *testing.T) {
	pods := make(map[string]*k8sV1.Pod)
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	for i := 0; i < numKubernetesWorkers; i++ {
		name := petName.Generate(3, "-")
		gracePeriod := int64(15)
		pod := &k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: "default"}}
		podInterface.On("Create", context.TODO(), pod, metaV1.CreateOptions{}).Return(
			&k8sV1.Pod{}, nil).Run(func(args mock.Arguments) { pods[name] = pod.DeepCopy() })
		podInterface.On("Delete", context.TODO(), name,
			metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod}).Return(nil).Run(
			func(args mock.Arguments) { delete(pods, name) })
		startMockPod(k8sRequestQueue, name, nil)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	createCancelled := false
	errorHandler := (errorCallbackFunc)(func(e error) {
		defer wg.Done()
		switch e := e.(type) {
		case resourceCreationCancelled:
			createCancelled = true
		default:
			panic(fmt.Sprintf("unexpected error %T", e))
		}
	})
	name := petName.Generate(3, "-")
	gracePeriod := int64(15)
	mockPod := startMockPod(k8sRequestQueue, name, &errorHandler)
	pod := &k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: "default"}}
	podInterface.On("Create", context.TODO(), pod, metaV1.CreateOptions{}).Return(
		&k8sV1.Pod{}, nil).Run(func(args mock.Arguments) { pods[name] = pod.DeepCopy() })
	podInterface.On("Delete", context.TODO(), name,
		metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod}).Return(nil).Run(
		func(args mock.Arguments) { delete(pods, name) })

	assert.Equal(t, createCancelled, false)
	mockPod.delete()
	wg.Wait()
	assert.Equal(t, createCancelled, true)
}

func TestRequestQueueCreationFailed(t *testing.T) {
	pods := make(map[string]*k8sV1.Pod)
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	var wg sync.WaitGroup
	wg.Add(1)
	createFailed := false
	errorHandler := (errorCallbackFunc)(func(e error) {
		defer wg.Done()
		switch e := e.(type) {
		case resourceCreationFailed:
			createFailed = true
		default:
			panic(fmt.Sprintf("unexpected error %T", e))
		}
	})
	name := petName.Generate(3, "-")
	gracePeriod := int64(15)
	mockPod := startMockPod(k8sRequestQueue, name, &errorHandler)
	pod := &k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: "default"}}
	podInterface.On("Create", context.TODO(), pod, metaV1.CreateOptions{}).Return(
		&k8sV1.Pod{}, nil).Run(func(args mock.Arguments) { pods[name] = pod.DeepCopy() })
	podInterface.On("Delete", context.TODO(), name,
		metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod}).Return(nil).Run(
		func(args mock.Arguments) { delete(pods, name) })

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, createFailed, false)

	mockPod.create()
	wg.Wait()
	assert.Equal(t, createFailed, true)
}

func TestRequestQueueDeletionFailed(t *testing.T) {
	pods := make(map[string]*k8sV1.Pod)
	podInterface := &mocks.PodInterface{}
	configMapInterface := &mockConfigMapInterface{configMaps: make(map[string]*k8sV1.ConfigMap)}

	k8sRequestQueue := startRequestQueue(
		map[string]typedV1.PodInterface{"default": podInterface},
		map[string]typedV1.ConfigMapInterface{"default": configMapInterface},
	)

	var wg sync.WaitGroup
	wg.Add(1)
	deleteFailed := false
	errorHandler := (errorCallbackFunc)(func(e error) {
		defer wg.Done()
		switch e := e.(type) {
		case resourceDeletionFailed:
			deleteFailed = true
		default:
			panic(fmt.Sprintf("unexpected error %T", e))
		}
	})
	name := petName.Generate(3, "-")
	gracePeriod := int64(15)
	mockPod := startMockPod(k8sRequestQueue, name, &errorHandler)
	pod := &k8sV1.Pod{ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: "default"}}
	podInterface.On("Create", context.TODO(), pod, metaV1.CreateOptions{}).Return(
		&k8sV1.Pod{}, nil).Run(func(args mock.Arguments) { pods[name] = pod.DeepCopy() })
	podInterface.On("Delete", context.TODO(), name,
		metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod}).Return(nil).Run(
		func(args mock.Arguments) { delete(pods, name) })

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, deleteFailed, false)

	mockPod.delete()

	waitForPendingRequestToFinish(k8sRequestQueue)
	assert.Equal(t, deleteFailed, false)

	mockPod.delete()
	wg.Wait()
	assert.Equal(t, deleteFailed, true)
}
