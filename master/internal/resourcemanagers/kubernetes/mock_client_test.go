package kubernetes

import (
	"sync"
	"time"

	"github.com/pkg/errors"

	k8sV1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type mockConfigMapInterface struct {
	configMaps map[string]*k8sV1.ConfigMap
	mux        sync.Mutex
}

func (m *mockConfigMapInterface) Create(cm *k8sV1.ConfigMap) (*k8sV1.ConfigMap, error) {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, present := m.configMaps[cm.Name]; present {
		return nil, errors.Errorf("configMap with name %s already exists", cm.Name)
	}

	m.configMaps[cm.Name] = cm.DeepCopy()
	return m.configMaps[cm.Name], nil
}

func (m *mockConfigMapInterface) Update(*k8sV1.ConfigMap) (*k8sV1.ConfigMap, error) {
	panic("implement me")
}

func (m *mockConfigMapInterface) Delete(name string, options *metaV1.DeleteOptions) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, present := m.configMaps[name]; !present {
		return errors.Errorf("configMap with name %s doesn't exists", name)
	}

	delete(m.configMaps, name)
	return nil
}

func (m *mockConfigMapInterface) DeleteCollection(
	options *metaV1.DeleteOptions,
	listOptions metaV1.ListOptions,
) error {
	panic("implement me")
}

func (m *mockConfigMapInterface) Get(
	name string,
	options metaV1.GetOptions,
) (*k8sV1.ConfigMap, error) {
	panic("implement me")
}

func (m *mockConfigMapInterface) List(opts metaV1.ListOptions) (*k8sV1.ConfigMapList, error) {
	panic("implement me")
}

func (m *mockConfigMapInterface) Watch(opts metaV1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (m *mockConfigMapInterface) Patch(
	name string,
	pt types.PatchType,
	data []byte,
	subresources ...string,
) (result *k8sV1.ConfigMap, err error) {
	panic("implement me")
}

type mockPodInterface struct {
	pods map[string]*k8sV1.Pod
	// Simulates latency of the real k8 API server.
	operationalDelay time.Duration
	mux              sync.Mutex
}

func (m *mockPodInterface) Create(pod *k8sV1.Pod) (*k8sV1.Pod, error) {
	time.Sleep(m.operationalDelay)
	m.mux.Lock()
	defer m.mux.Unlock()

	if _, present := m.pods[pod.Name]; present {
		return nil, errors.Errorf("pod with name %s already exists", pod.Name)
	}

	m.pods[pod.Name] = pod.DeepCopy()
	return m.pods[pod.Name], nil
}

func (m *mockPodInterface) Update(*k8sV1.Pod) (*k8sV1.Pod, error) {
	panic("implement me")
}

func (m *mockPodInterface) UpdateStatus(*k8sV1.Pod) (*k8sV1.Pod, error) {
	panic("implement me")
}

func (m *mockPodInterface) Delete(name string, options *metaV1.DeleteOptions) error {
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
	options *metaV1.DeleteOptions,
	listOptions metaV1.ListOptions,
) error {
	panic("implement me")
}

func (m *mockPodInterface) Get(name string, options metaV1.GetOptions) (*k8sV1.Pod, error) {
	panic("implement me")
}

func (m *mockPodInterface) List(opts metaV1.ListOptions) (*k8sV1.PodList, error) {
	time.Sleep(m.operationalDelay)
	m.mux.Lock()
	defer m.mux.Unlock()

	podList := &k8sV1.PodList{}
	for _, pod := range m.pods {
		podList.Items = append(podList.Items, *pod)
	}

	return podList, nil
}

func (m *mockPodInterface) Watch(opts metaV1.ListOptions) (watch.Interface, error) {
	panic("implement me")
}

func (m *mockPodInterface) Patch(
	name string,
	pt types.PatchType,
	data []byte,
	subresources ...string,
) (result *k8sV1.Pod, err error) {
	panic("implement me")
}

func (m *mockPodInterface) GetEphemeralContainers(
	podName string,
	options metaV1.GetOptions,
) (*k8sV1.EphemeralContainers, error) {
	panic("implement me")
}

func (m *mockPodInterface) UpdateEphemeralContainers(
	podName string,
	ephemeralContainers *k8sV1.EphemeralContainers,
) (*k8sV1.EphemeralContainers, error) {
	panic("implement me")
}

func (m *mockPodInterface) Bind(binding *k8sV1.Binding) error {
	panic("implement me")
}

func (m *mockPodInterface) Evict(eviction *v1beta1.Eviction) error {
	panic("implement me")
}

func (m *mockPodInterface) GetLogs(name string, opts *k8sV1.PodLogOptions) *rest.Request {
	panic("implement me")
}
