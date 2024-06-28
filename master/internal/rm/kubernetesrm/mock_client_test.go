package kubernetesrm

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/pkg/errors"

	batchV1 "k8s.io/api/batch/v1"
	k8sV1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/api/policy/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	applyBatchV1 "k8s.io/client-go/applyconfigurations/batch/v1"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/rest"
)

/* As of Aug. 7, 2023, the mock interfaces found
here & used in pod_test.go cannot be replaced easily
by mockery-generated mocks without overcomplicating the code
& its readability. In pod_test.go, the tests send messages
to the Actor system, which are dealt with by a mock
receiver struct. The mockery-generated interfaces do not
execute the necessary receiver-related code, unlike the mocks here. */

type mockConfigMapInterface struct {
	configMaps map[string]*k8sV1.ConfigMap
	mux        sync.Mutex
}

func (m *mockConfigMapInterface) Create(
	ctx context.Context, cm *k8sV1.ConfigMap, opts metaV1.CreateOptions,
) (*k8sV1.ConfigMap, error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, present := m.configMaps[cm.Name]; present {
		return nil, errors.Errorf("configMap with name %s already exists", cm.Name)
	}

	m.configMaps[cm.Name] = cm.DeepCopy()
	return m.configMaps[cm.Name], nil
}

func (m *mockConfigMapInterface) Update(
	context.Context, *k8sV1.ConfigMap, metaV1.UpdateOptions,
) (*k8sV1.ConfigMap, error) {
	panic("implement me")
}

func (m *mockConfigMapInterface) Delete(
	ctx context.Context, name string, options metaV1.DeleteOptions,
) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, present := m.configMaps[name]; !present {
		return errors.Errorf("configMap with name %s doesn't exists", name)
	}

	delete(m.configMaps, name)
	return nil
}

func (m *mockConfigMapInterface) DeleteCollection(
	ctx context.Context, options metaV1.DeleteOptions, listOptions metaV1.ListOptions,
) error {
	panic("implement me")
}

func (m *mockConfigMapInterface) Get(
	ctx context.Context, name string, options metaV1.GetOptions,
) (*k8sV1.ConfigMap, error) {
	panic("implement me")
}

func (m *mockConfigMapInterface) List(
	ctx context.Context, opts metaV1.ListOptions,
) (*k8sV1.ConfigMapList, error) {
	panic("implement me")
}

func (m *mockConfigMapInterface) Watch(
	ctx context.Context, opts metaV1.ListOptions,
) (watch.Interface, error) {
	panic("implement me")
}

func (m *mockConfigMapInterface) Patch(
	ctx context.Context, name string, pt types.PatchType, data []byte, opts metaV1.PatchOptions,
	subresources ...string,
) (result *k8sV1.ConfigMap, err error) {
	panic("implement me")
}

func (m *mockConfigMapInterface) Apply(
	ctx context.Context, configMap *corev1.ConfigMapApplyConfiguration, opts metaV1.ApplyOptions,
) (result *k8sV1.ConfigMap, err error) {
	panic("implement me")
}

type mockPodInterface struct {
	pods map[string]*k8sV1.Pod
	// Simulates latency of the real k8 API server.
	operationalDelay time.Duration
	logMessage       *string
	mux              sync.Mutex
}

func (m *mockPodInterface) Create(
	ctx context.Context, pod *k8sV1.Pod, opts metaV1.CreateOptions,
) (*k8sV1.Pod, error) {
	time.Sleep(m.operationalDelay)
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, present := m.pods[pod.Name]; present {
		return nil, errors.Errorf("pod with name %s already exists", pod.Name)
	}

	m.pods[pod.Name] = pod.DeepCopy()
	return m.pods[pod.Name], nil
}

func (m *mockPodInterface) Update(
	context.Context, *k8sV1.Pod, metaV1.UpdateOptions,
) (*k8sV1.Pod, error) {
	panic("implement me")
}

func (m *mockPodInterface) UpdateStatus(
	context.Context, *k8sV1.Pod, metaV1.UpdateOptions,
) (*k8sV1.Pod, error) {
	panic("implement me")
}

func (m *mockPodInterface) Delete(
	ctx context.Context, name string, options metaV1.DeleteOptions,
) error {
	time.Sleep(m.operationalDelay)
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, present := m.pods[name]; !present {
		return errors.Errorf("pod with name %s doesn't exists", name)
	}

	delete(m.pods, name)
	return nil
}

func (m *mockPodInterface) DeleteCollection(
	ctx context.Context, options metaV1.DeleteOptions, listOptions metaV1.ListOptions,
) error {
	panic("implement me")
}

func (m *mockPodInterface) Get(
	ctx context.Context, name string, options metaV1.GetOptions,
) (*k8sV1.Pod, error) {
	panic("implement me")
}

func (m *mockPodInterface) List(
	ctx context.Context, opts metaV1.ListOptions,
) (*k8sV1.PodList, error) {
	time.Sleep(m.operationalDelay)
	m.mux.Lock()
	defer m.mux.Unlock()

	podList := &k8sV1.PodList{}
	for _, pod := range m.pods {
		podList.Items = append(podList.Items, *pod)
	}

	return podList, nil
}

func (m *mockPodInterface) Watch(
	ctx context.Context, opts metaV1.ListOptions,
) (watch.Interface, error) {
	panic("implement me")
}

func (m *mockPodInterface) Patch(
	ctx context.Context, name string, pt types.PatchType, data []byte, opts metaV1.PatchOptions,
	subresources ...string,
) (result *k8sV1.Pod, err error) {
	panic("implement me")
}

func (m *mockPodInterface) Apply(
	ctx context.Context, pod *corev1.PodApplyConfiguration, opts metaV1.ApplyOptions,
) (result *k8sV1.Pod, err error) {
	panic("implement me")
}

func (m *mockPodInterface) ApplyStatus(
	ctx context.Context, pod *corev1.PodApplyConfiguration, opts metaV1.ApplyOptions,
) (result *k8sV1.Pod, err error) {
	panic("implement me")
}

func (m *mockPodInterface) UpdateEphemeralContainers(
	ctx context.Context, podName string, ephemeralContainers *k8sV1.Pod,
	opts metaV1.UpdateOptions,
) (*k8sV1.Pod, error) {
	panic("implement me")
}

func (m *mockPodInterface) Bind(context.Context, *k8sV1.Binding, metaV1.CreateOptions) error {
	panic("implement me")
}

func (m *mockPodInterface) Evict(ctx context.Context, eviction *v1beta1.Eviction) error {
	panic("implement me")
}

func (m *mockPodInterface) EvictV1(ctx context.Context, eviction *policyv1.Eviction) error {
	panic("implement me")
}

func (m *mockPodInterface) EvictV1beta1(ctx context.Context, eviction *v1beta1.Eviction) error {
	panic("implement me")
}

func (m *mockPodInterface) GetLogs(name string, opts *k8sV1.PodLogOptions) *rest.Request {
	client := cleanhttp.DefaultClient()
	client.Transport = &mockRoundTripInterface{message: m.logMessage}
	return rest.NewRequestWithClient(&url.URL{}, "", rest.ClientContentConfig{}, client)
}

func (m *mockPodInterface) ProxyGet(
	string, string, string, string, map[string]string,
) rest.ResponseWrapper {
	panic("implement me")
}

type mockRoundTripInterface struct {
	message *string
}

func (m *mockRoundTripInterface) RoundTrip(req *http.Request) (*http.Response, error) {
	var msg string
	if m.message != nil {
		msg = *m.message
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(msg)),
	}, nil
}

type mockJobInterface struct {
	jobs map[string]*batchV1.Job
	// Simulates latency of the real k8 API server.
	operationalDelay time.Duration
	mux              sync.Mutex
}

func (m *mockJobInterface) Create(
	ctx context.Context, job *batchV1.Job, opts metaV1.CreateOptions,
) (*batchV1.Job, error) {
	time.Sleep(m.operationalDelay)
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, present := m.jobs[job.Name]; present {
		return nil, errors.Errorf("pod with name %s already exists", job.Name)
	}

	m.jobs[job.Name] = job.DeepCopy()
	return m.jobs[job.Name], nil
}

func (m *mockJobInterface) Update(
	context.Context, *batchV1.Job, metaV1.UpdateOptions,
) (*batchV1.Job, error) {
	panic("implement me")
}

func (m *mockJobInterface) UpdateStatus(
	context.Context, *batchV1.Job, metaV1.UpdateOptions,
) (*batchV1.Job, error) {
	panic("implement me")
}

func (m *mockJobInterface) Delete(
	ctx context.Context, name string, options metaV1.DeleteOptions,
) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, present := m.jobs[name]; !present {
		return errors.Errorf("job with name %s doesn't exists", name)
	}

	delete(m.jobs, name)
	return nil
}

func (m *mockJobInterface) DeleteCollection(
	ctx context.Context, options metaV1.DeleteOptions, listOptions metaV1.ListOptions,
) error {
	panic("implement me")
}

func (m *mockJobInterface) Get(
	ctx context.Context, name string, options metaV1.GetOptions,
) (*batchV1.Job, error) {
	panic("implement me")
}

func (m *mockJobInterface) List(
	ctx context.Context, opts metaV1.ListOptions,
) (*batchV1.JobList, error) {
	time.Sleep(m.operationalDelay)
	m.mux.Lock()
	defer m.mux.Unlock()

	jobList := &batchV1.JobList{}
	for _, job := range m.jobs {
		jobList.Items = append(jobList.Items, *job)
	}

	return jobList, nil
}

func (m *mockJobInterface) Watch(
	ctx context.Context, opts metaV1.ListOptions,
) (watch.Interface, error) {
	panic("implement me")
}

func (m *mockJobInterface) Patch(
	ctx context.Context, name string, pt types.PatchType, data []byte, opts metaV1.PatchOptions,
	subresources ...string,
) (result *batchV1.Job, err error) {
	panic("implement me")
}

func (m *mockJobInterface) Apply(
	ctx context.Context, job *applyBatchV1.JobApplyConfiguration, opts metaV1.ApplyOptions,
) (result *batchV1.Job, err error) {
	panic("implement me")
}

func (m *mockJobInterface) ApplyStatus(
	ctx context.Context, job *applyBatchV1.JobApplyConfiguration, opts metaV1.ApplyOptions,
) (result *batchV1.Job, err error) {
	panic("implement me")
}
