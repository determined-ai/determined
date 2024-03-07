//go:build integration
// +build integration

package internal

import (
	"testing"

	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
)

func getMockResourceManager(poolName string) *mocks.ResourceManager {
	rm := &mocks.ResourceManager{}
	rm.On("ResolveResourcePool", "/", 0, 1).Return(poolName, nil)
	rm.On("ValidateResources", sproto.ValidateResourcesRequest{
		ResourcePool: poolName,
		Slots:        1,
		IsSingleNode: true,
	}).Return(sproto.ValidateResourcesResponse{}, nil, nil)
	return rm
}

func TestResolveResources(t *testing.T) {
	tests := map[string]struct {
		expectedPoolName string
		resourcePool     string
		slots            int
		workspaceID      int
	}{
		"basicTestCase": {
			expectedPoolName: "poolName",
			resourcePool:     "/",
			slots:            1,
			workspaceID:      0,
		},
	}

	for testCase, testVars := range tests {
		t.Run(testCase, func(t *testing.T) {
			m := &Master{
				rm:     getMockResourceManager(testVars.expectedPoolName),
				config: config.DefaultConfig(),
			}
			poolName, _, err := m.ResolveResources(testVars.resourcePool, testVars.slots, testVars.workspaceID, true)

			require.NoError(t, err, "Error in ResolveResources()")
			require.Equal(t, testVars.expectedPoolName, poolName)
		})
	}
}

func TestFillTaskSpec(t *testing.T) {
	tests := map[string]struct {
		poolName       string
		agentUserGroup *model.AgentUserGroup
		userModel      *model.User
		workDir        string
	}{
		"basicTestCase": {
			poolName:       "poolName",
			agentUserGroup: &model.AgentUserGroup{},
			userModel:      &model.User{},
			workDir:        "/",
		},
	}
	for testCase, testVars := range tests {
		t.Run(testCase, func(t *testing.T) {
			rm := getMockResourceManager(testVars.poolName)
			m := &Master{
				rm:       rm,
				config:   config.DefaultConfig(),
				taskSpec: &tasks.TaskSpec{},
			}
			expectedTaskSpec := tasks.TaskSpec{
				TaskContainerDefaults: model.TaskContainerDefaultsConfig{
					WorkDir: &testVars.workDir,
				},
				AgentUserGroup: testVars.agentUserGroup,
				Owner:          testVars.userModel,
			}
			rm.On("TaskContainerDefaults",
				testVars.poolName,
				m.config.TaskContainerDefaults,
			).Return(model.TaskContainerDefaultsConfig{WorkDir: &testVars.workDir}, nil)
			taskSpec, err := m.fillTaskSpec(testVars.poolName, testVars.agentUserGroup, testVars.userModel)
			require.NoError(t, err, "Error in fillTaskSpec()")
			require.Equal(t, expectedTaskSpec, taskSpec)
		})
	}
}

func TestFillTaskConfigPodSpec(t *testing.T) {
	taskSpec := tasks.TaskSpec{
		TaskContainerDefaults: model.TaskContainerDefaultsConfig{
			CPUPodSpec: &k8sV1.Pod{
				TypeMeta: metaV1.TypeMeta{Kind: "cpu"},
			},
			GPUPodSpec: &k8sV1.Pod{
				TypeMeta: metaV1.TypeMeta{Kind: "gpu"},
			},
		},
	}
	tests := map[string]struct {
		poolName            string
		slots               int
		taskSpec            tasks.TaskSpec
		expectedPodSpecKind string
	}{
		"CPUPodSpec": {
			poolName:            "poolName",
			slots:               0,
			taskSpec:            taskSpec,
			expectedPodSpecKind: "cpu",
		},
		"GPUPodSpec": {
			poolName:            "poolName",
			slots:               2,
			taskSpec:            taskSpec,
			expectedPodSpecKind: "gpu",
		},
	}

	for testCase, testVars := range tests {
		t.Run(testCase, func(t *testing.T) {
			env := &model.Environment{
				PodSpec: &k8sV1.Pod{},
			}
			fillTaskConfig(testVars.slots, testVars.taskSpec, env)
			require.Equal(t, testVars.expectedPodSpecKind, env.PodSpec.TypeMeta.Kind)
		})
	}
}

func TestFillContextDir(t *testing.T) {
	tests := map[string]struct {
		defaultWorkDir   string
		contextDirectory []*utilv1.File
	}{
		"basicTestCase": {
			defaultWorkDir:   "/",
			contextDirectory: []*utilv1.File{{Content: []byte{1}}},
		},
	}
	for testCase, testVars := range tests {
		t.Run(testCase, func(t *testing.T) {
			var configWorkDir *string

			userFiles := filesToArchive(testVars.contextDirectory)
			expectedBytes, err := archive.ToTarGz(userFiles)
			require.NoError(t, err, "Error in ToTarGz() for TestFillContexxtDir")

			var contextDirectoryBytes []byte
			_, contextDirectoryBytes, err = fillContextDir(
				configWorkDir,
				&testVars.defaultWorkDir,
				testVars.contextDirectory,
			)
			require.NoError(t, err, "Error in fillContextDir()")
			require.Equal(t, expectedBytes, contextDirectoryBytes)
		})
	}
}

func TestGetTaskSessionToken(t *testing.T) {
	_, userModel, ctx := setupAPITest(t, nil)

	token, err := getTaskSessionToken(ctx, &userModel)
	require.NoError(t, err, "Error in getTaskSessionToken()")
	require.Greater(t, len(token), 0)
}
