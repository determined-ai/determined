//go:build integration
// +build integration

package tasks

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
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

	for testCase, testVars := range tests {
		t.Run(testCase, func(t *testing.T) {
			err := etc.SetRootPath("../../static/srv/")
			require.NoError(t, err)
			res := testVars.taskSpec.ToTaskSpec()
			require.Equal(t, testVars.expectedDescription, res.Description)
			require.Equal(t, testVars.expectedSlots, *res.ResourcesConfig.RawSlots)
			require.Equal(t, testVars.expectedIsSingleNode, *res.ResourcesConfig.RawIsSingleNode)
			require.Equal(t, testVars.expectedEntrypoint, res.Entrypoint[0])
			require.Equal(t, testVars.expectedType, res.TaskType)
		})
	}
}
