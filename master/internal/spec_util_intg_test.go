//go:build integration
// +build integration

package internal

import (
	"testing"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/utilv1"
	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getMockResourceManager(poolName string) *mocks.ResourceManager {
	rm := &mocks.ResourceManager{}
	rm.On("ResolveResourcePool", "/", 0, 1).Return(poolName, nil)
	rm.On("ValidateResourcePoolAvailability", &sproto.ValidateResourcePoolAvailabilityRequest{
		Name:  poolName,
		Slots: 1,
	}).Return(nil, nil)
	rm.On("ValidateResources", poolName, 1, true).Return(nil)
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

	for test_case, test_vars := range tests {
		t.Run(test_case, func(t *testing.T) {
			m :=
				&Master{
					rm:     getMockResourceManager(test_vars.expectedPoolName),
					config: config.DefaultConfig(),
				}
			poolName, _, err := m.ResolveResources(test_vars.resourcePool, test_vars.slots, test_vars.workspaceID)

			require.NoError(t, err, "Error in ResolveResources()")
			require.Equal(t, test_vars.expectedPoolName, poolName)
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
	for test_case, test_vars := range tests {
		t.Run(test_case, func(t *testing.T) {
			rm := getMockResourceManager(test_vars.poolName)
			m := &Master{
				rm:       rm,
				config:   config.DefaultConfig(),
				taskSpec: &tasks.TaskSpec{},
			}
			expectedTaskSpec := tasks.TaskSpec{
				TaskContainerDefaults: model.TaskContainerDefaultsConfig{
					WorkDir: &test_vars.workDir,
				},
				AgentUserGroup: test_vars.agentUserGroup,
				Owner:          test_vars.userModel,
			}
			rm.On("TaskContainerDefaults", test_vars.poolName, m.config.TaskContainerDefaults).Return(model.TaskContainerDefaultsConfig{WorkDir: &test_vars.workDir}, nil)
			taskSpec, err := m.fillTaskSpec(test_vars.poolName, test_vars.agentUserGroup, test_vars.userModel)
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

	for test_case, test_vars := range tests {
		t.Run(test_case, func(t *testing.T) {
			env := &model.Environment{
				PodSpec: &k8sV1.Pod{},
			}
			fillTaskConfig(test_vars.slots, test_vars.taskSpec, env)
			require.Equal(t, test_vars.expectedPodSpecKind, env.PodSpec.TypeMeta.Kind)
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
	for test_case, test_vars := range tests {
		t.Run(test_case, func(t *testing.T) {
			var configWorkDir *string

			userFiles := filesToArchive(test_vars.contextDirectory)
			expectedBytes, err := archive.ToTarGz(userFiles)
			require.NoError(t, err, "Error in ToTarGz() for TestFillContexxtDir")

			var contextDirectoryBytes []byte
			configWorkDir, contextDirectoryBytes, err = fillContextDir(configWorkDir, &test_vars.defaultWorkDir, test_vars.contextDirectory)
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
