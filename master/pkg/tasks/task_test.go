package tasks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	k8sV1 "k8s.io/api/core/v1"

	"github.com/determined-ai/determined/master/pkg/archive"
	"github.com/determined-ai/determined/master/pkg/cproto"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestTaskSpecClone(t *testing.T) {
	//nolint:exhaustruct
	orig := &TaskSpec{
		Environment: expconf.EnvironmentConfig{
			RawPodSpec: &expconf.PodSpec{
				Spec: k8sV1.PodSpec{
					ServiceAccountName: "test",
				},
			},
		},
		ExtraEnvVars: map[string]string{"a": "true"},
	}

	cloned, err := orig.Clone()
	require.NoError(t, err)
	require.Equal(t, orig, cloned)

	// Actually deep cloned.
	orig.ExtraEnvVars["a"] = "diff"
	require.Equal(t, map[string]string{"a": "true"}, cloned.ExtraEnvVars)
}

// finds the first startup hook.
func findFirstStartupHook(runArchives []cproto.RunArchive) *archive.Item {
	for _, runArchive := range runArchives {
		for _, item := range runArchive.Archive {
			if strings.HasSuffix(item.Path, StartupHookScript) {
				return &item
			}
		}
	}
	return nil
}

func TestTCDStartupHook(t *testing.T) {
	err := etc.SetRootPath("../../static/srv")
	require.NoError(t, err)
	tcd := model.TaskContainerDefaultsConfig{}
	taskSpec := TaskSpec{
		TaskContainerDefaults: tcd,
		AgentUserGroup:        &model.AgentUserGroup{},
	}
	userArchives, _ := taskSpec.Archives()
	hook := findFirstStartupHook(userArchives)
	require.Nil(t, hook, "Empty TCD should not generate a startup hook file")

	entryPoints := taskSpec.tCDStartupEntrypoint()
	require.Empty(t, entryPoints, "Empty TCD should not have startup hook")
	allEntrypoints := taskSpec.CombinedEntrypoint()
	require.NotContains(t, allEntrypoints, "--tcd_startup_hook_filename", "Empty TCD should not have startup hook")
	require.NotContains(t, allEntrypoints, taskSpec.tcdStartHookPath(), "Empty TCD should not have startup hook")

	// with a startup hook.

	tcd.StartupHook = "echo hi"
	taskSpec.TaskContainerDefaults = tcd
	userArchives, _ = taskSpec.Archives()
	hook = findFirstStartupHook(userArchives)
	require.NotNil(t, hook, "TCD with startup hook should generate a startup hook file")
	require.Contains(t, string(hook.Content), "echo hi")
	entryPoints = taskSpec.tCDStartupEntrypoint()
	require.NotEmpty(t, entryPoints, "TCD with startup hook should have startup hook")
	require.Len(t, entryPoints, 2, "TCD with startup hook should have two parts")
	allEntrypoints = taskSpec.CombinedEntrypoint()
	require.Contains(t, allEntrypoints, "--tcd_startup_hook_filename", "TCD with startup hook should have startup hook")
	require.Contains(t, allEntrypoints, taskSpec.tcdStartHookPath(), "TCD with startup hook should have startup hook")
}
