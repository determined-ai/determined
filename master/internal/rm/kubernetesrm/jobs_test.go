package kubernetesrm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	k8sV1 "k8s.io/api/core/v1"

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

	clientSet := &mocks.K8sClientsetInterface{}
	coreV1 := &mocks.K8sCoreV1Interface{}
	namespaces := &mocks.NamespaceInterface{}

	clientSet.On("CoreV1").Maybe().Return(coreV1)
	coreV1.On("Namespaces").Maybe().Return(namespaces)

	const (
		existantNamespaceName    = "already-exists"
		nonexistantNamespaceName = "good-to-go"
	)

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
	).Once().Return(errors.New("this namespace already exists"))
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
	).Once().Return(nil)

	j := jobsService{
		clientSet: clientSet,
	}

	err := j.createNamespace(existantNamespaceName)
	assert.Error(t, err, "expected error when creating already-existing namespace")
	// TODO: verify interfaces

	err = j.createNamespace(nonexistantNamespaceName)
	assert.NoError(t, err, "unexpected error when creating namespace")
	// TODO: verify interfaces
}
