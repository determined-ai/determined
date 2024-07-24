package kubernetesrm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	k8sBatchV1 "k8s.io/api/batch/v1"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	typedBatchV1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	typedV1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/determined-ai/determined/master/internal/mocks"
)

func TestCreateNamespace(t *testing.T) {
	// Expected behavior:
	// - Creates the namespace by calling CoreV1().Namespaces().Create()
	//     - Returns proper error when the namespace already exists
	// - Stores clients for the new namespace:
	//     - podInterfaces
	//     - configMapInterfaces
	//     - jobInterfaces

	// TODO:
	// - Mock out Namespaces object
	//     - Generate with Mockery?
	// - .expect() Namespaces' Create() method
	// - Consider changing the method to do better error handling
	// - After each test case, make sure the clients are the expected new ones

	// Build some mocks before configuring how they're linked
	clientSet := &mocks.K8sClientsetInterface{}
	coreV1 := &mocks.K8sCoreV1Interface{}
	batchV1 := &mocks.K8sBatchV1Interface{}
	namespaces := &mocks.NamespaceInterface{}
	pod := &mocks.PodInterface{}
	configMaps := &mocks.ConfigMapInterface{}
	jobs := &mocks.JobInterface{}
	events := &mocks.EventInterface{}

	// We eventually want to make sure each one's expectations were met, so we'll use this
	// to check each one.
	type AssertExpectationser interface {
		AssertExpectations(mock.TestingT) bool
	}
	mocksToCheck := []AssertExpectationser{coreV1, batchV1, namespaces, pod, configMaps, jobs}
	defer func() {
		for _, m := range mocksToCheck {
			m.AssertExpectations(t)
		}
	}()

	const (
		existantNamespaceName    = "already-exists"
		nonexistantNamespaceName = "good-to-go"
	)

	clientSet.On("CoreV1").Maybe().Return(coreV1)
	clientSet.On("BatchV1").Return(batchV1)
	coreV1.On("Namespaces").Return(namespaces)
	coreV1.On("Pods", nonexistantNamespaceName).Return(pod)
	coreV1.On("ConfigMaps", nonexistantNamespaceName).Return(configMaps)
	coreV1.On("Events", nonexistantNamespaceName).Return(events)
	batchV1.On("Jobs", nonexistantNamespaceName).Return(jobs)
	events.On("List", mock.Anything, mock.Anything).Return(&k8sV1.EventList{
		Items: []k8sV1.Event{},
		ListMeta: metaV1.ListMeta{
			ResourceVersion: "1",
		},
	}, nil)
	events.On("Watch", mock.Anything, mock.Anything).Return(&mockWatcher{}, nil)
	pod.On("List", mock.Anything, mock.Anything).Return(&k8sV1.PodList{
		Items: []k8sV1.Pod{},
		ListMeta: metaV1.ListMeta{
			ResourceVersion: "1",
		},
	}, nil)
	pod.On("Watch", mock.Anything, mock.Anything).Return(&mockWatcher{}, nil)
	jobs.On("List", mock.Anything, mock.Anything).Return(&k8sBatchV1.JobList{}, nil)

	// Expect a call trying to create the already-existing namespace
	namespaces.On("Create",
		mock.Anything,
		mock.MatchedBy(func(ns *k8sV1.Namespace) bool {
			if ns.ObjectMeta.Name == existantNamespaceName {
				return true
			}
			return false
		}),
		mock.Anything,
	).Once().Return(nil, errors.New("this namespace already exists"))
	// Expect a call trying to create the not already-existing namespace
	namespaces.On("Create",
		mock.Anything,
		mock.MatchedBy(func(ns *k8sV1.Namespace) bool {
			if ns.ObjectMeta.Name == nonexistantNamespaceName {
				return true
			}
			return false
		}),
		mock.Anything,
	).Once().Return(nil, nil)

	j := jobsService{
		clientSet:               clientSet,
		podInterfaces:           map[string]typedV1.PodInterface{},
		configMapInterfaces:     map[string]typedV1.ConfigMapInterface{},
		jobInterfaces:           map[string]typedBatchV1.JobInterface{},
		namespacesWithInformers: map[string]bool{},
	}
	err := j.createNamespace(existantNamespaceName)
	assert.Error(t, err, "expected error when creating already-existing namespace")
	// TODO: verify interfaces

	j = jobsService{
		clientSet:               clientSet,
		podInterfaces:           map[string]typedV1.PodInterface{},
		configMapInterfaces:     map[string]typedV1.ConfigMapInterface{},
		jobInterfaces:           map[string]typedBatchV1.JobInterface{},
		namespacesWithInformers: map[string]bool{},
	}
	err = j.createNamespace(nonexistantNamespaceName)
	assert.NoError(t, err, "unexpected error when creating namespace")
	// TODO: verify interfaces
}
