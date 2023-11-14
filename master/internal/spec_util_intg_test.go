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
	expectedPoolName := "poolName"
	api := &apiServer{
		m: &Master{
			rm:     getMockResourceManager(expectedPoolName),
			config: config.DefaultConfig(),
		},
	}
	resourcePool := "/"
	slots := 1
	workspaceID := 0

	poolName, _, err := ResolveResources(api, resourcePool, slots, workspaceID)

	require.NoError(t, err, "Error in ResolveResources()")
	require.Equal(t, expectedPoolName, poolName)
}

func TestFillTaskSpec(t *testing.T) {
	poolName := "poolName"
	rm := getMockResourceManager(poolName)
	api := &apiServer{
		m: &Master{
			rm:       rm,
			config:   config.DefaultConfig(),
			taskSpec: &tasks.TaskSpec{},
		},
	}
	workDir := "/"
	rm.On("TaskContainerDefaults", poolName, api.m.config.TaskContainerDefaults).Return(model.TaskContainerDefaultsConfig{WorkDir: &workDir}, nil)
	agentUserGroup := &model.AgentUserGroup{}
	userModel := &model.User{}
	expectedTaskSpec := tasks.TaskSpec{
		TaskContainerDefaults: model.TaskContainerDefaultsConfig{
			WorkDir: &workDir,
		},
		AgentUserGroup: agentUserGroup,
		Owner:          userModel,
	}
	taskSpec, err := fillTaskSpec(api, poolName, agentUserGroup, userModel)
	require.NoError(t, err, "Error in fillTaskSpec()")
	require.Equal(t, expectedTaskSpec, taskSpec)

}

func TestFillTaskConfigPodSpec(t *testing.T) {
	poolName := "poolName"
	slots := 0

	var resourcePoolDest *string
	var resourceSlotsDest *int

	cpu_spec := &k8sV1.Pod{
		TypeMeta: metaV1.TypeMeta{Kind: "cpu"},
	}
	gpu_spec := &k8sV1.Pod{
		TypeMeta: metaV1.TypeMeta{Kind: "gpu"},
	}
	taskSpec := tasks.TaskSpec{
		TaskContainerDefaults: model.TaskContainerDefaultsConfig{
			CPUPodSpec: cpu_spec,
			GPUPodSpec: gpu_spec,
		},
	}
	env := &model.Environment{
		PodSpec: &k8sV1.Pod{},
	}
	fillTaskConfig(&resourcePoolDest, poolName, &resourceSlotsDest, slots, taskSpec, env)
	require.Equal(t, cpu_spec.TypeMeta.Kind, env.PodSpec.TypeMeta.Kind)
	require.Equal(t, poolName, *resourcePoolDest)
	require.Equal(t, slots, *resourceSlotsDest)

	slots = 2
	env = &model.Environment{
		PodSpec: &k8sV1.Pod{},
	}
	fillTaskConfig(&resourcePoolDest, poolName, &resourceSlotsDest, slots, taskSpec, env)
	require.Equal(t, gpu_spec.TypeMeta.Kind, env.PodSpec.TypeMeta.Kind)
}

func TestFillContextDir(t *testing.T) {
	var configWorkDirDest *string
	defaultWorkDir := "/"
	contextDirectory := []*utilv1.File{{Content: []byte{1}}}

	userFiles := filesToArchive(contextDirectory)
	expectedBytes, err := archive.ToTarGz(userFiles)
	require.NoError(t, err, "Error in ToTarGz() for TestFillContexxtDir")

	var contextDirectoryBytes []byte
	contextDirectoryBytes, err = fillContextDir(&configWorkDirDest, &defaultWorkDir, contextDirectory)
	require.NoError(t, err, "Error in fillContextDir()")
	require.Equal(t, expectedBytes, contextDirectoryBytes)
}

func TestGetTaskSessionToken(t *testing.T) {
	_, userModel, ctx := setupAPITest(t, nil)

	token, err := getTaskSessionToken(ctx, &userModel)
	require.NoError(t, err, "Error in getTaskSessionToken()")
	require.Greater(t, len(token), 0)
}
