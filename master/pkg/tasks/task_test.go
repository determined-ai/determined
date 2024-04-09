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

	// with a startup hook.

	tcd.StartupHook = "echo hi"
	taskSpec.TaskContainerDefaults = tcd
	userArchives, _ = taskSpec.Archives()
	hook = findFirstStartupHook(userArchives)
	require.NotNil(t, hook, "TCD with startup hook should generate a startup hook file")
	require.Contains(t, string(hook.Content), "echo hi")
}
