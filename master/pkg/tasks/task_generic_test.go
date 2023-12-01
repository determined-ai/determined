//go:build integration
// +build integration

package tasks

import (
	"testing"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/stretchr/testify/require"
)

func TestToTaskSpec(t *testing.T) {
	tests := map[string]struct {
		expectedType         model.TaskType
		expectedEntrypoint   string
		expectedSlots        int
		expectedIsSingleNode bool
		expectedDescription  string
		taskSpec             GenericTaskSpec
	}{
		"basicTestCase": {
			expectedDescription:  "generic-task",
			expectedSlots:        1,
			expectedIsSingleNode: true,
			expectedEntrypoint:   "/run/determined/generic-task-entrypoint.sh",
			expectedType:         model.TaskTypeGeneric,
			taskSpec: GenericTaskSpec{
				GenericTaskConfig: model.DefaultConfigGenericTaskConfig(&model.TaskContainerDefaultsConfig{WorkDir: ptrs.Ptr("/")}),
			},
		},
	}

	for test_case, test_vars := range tests {
		t.Run(test_case, func(t *testing.T) {
			etc.SetRootPath("../../static/srv/")
			res := test_vars.taskSpec.ToTaskSpec()
			require.Equal(t, test_vars.expectedDescription, res.Description)
			require.Equal(t, test_vars.expectedSlots, *res.ResourcesConfig.RawSlots)
			require.Equal(t, test_vars.expectedIsSingleNode, *res.ResourcesConfig.RawIsSingleNode)
			require.Equal(t, test_vars.expectedEntrypoint, res.Entrypoint[0])
			require.Equal(t, test_vars.expectedType, res.TaskType)
		})
	}
}
