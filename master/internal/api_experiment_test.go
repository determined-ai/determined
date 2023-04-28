package internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExperimentSearchApiFilterParsing(t *testing.T) {
	invalidTestCases := []string{
		// No operator specified in field
		`{"children":[{"columnName":"resourcePool","kind":"field","value":"default"}],"conjunction":"and","kind":"group"}`,

		// No conjunction in group
		`{"children":[{"columnName":"resourcePool","kind":"field","operator":"=","value":"default"}],"kind":"group"}`,

		// invalid group conjunction
		`{"children":[{"columnName":"resourcePool","kind":"field","operator":"=","value":"default"}],"conjunction":"invalid","kind":"group"}`,

		// invalid operator
		`{"children":[{"columnName":"resourcePool","kind":"field","operator":"invalid","value":"default"}],"conjunction":"and","kind":"group"}`,
	}
	for _, c := range invalidTestCases {
		var experimentFilter ExperimentFilter
		err := json.Unmarshal([]byte(c), &experimentFilter)
		require.NoError(t, err)
		_, err = experimentFilter.toSql()
		require.Error(t, err)
	}
	validTestCases := [][2]string{
		// TOOD add more valid test cases
		{`{"children":[{"children":[],"conjunction":"and","id":"e8f7edb2-e067-49f8-96a3-024684a8fcf3","kind":"group"},{"children":[],"conjunction":"and","id":"0f19eed1-f6a2-4123-a9be-e5688fa7ba98","kind":"group"},{"children":[],"conjunction":"and","id":"62bed64f-9b90-4411-8837-8376fa3df51a","kind":"group"}],"conjunction":"or","id":"ROOT","kind":"group"}`, `(true OR true OR true)`},
		{`{"children":[{"children":[],"conjunction":"and","id":"e8f7edb2-e067-49f8-96a3-024684a8fcf3","kind":"group"},{"children":[],"conjunction":"and","id":"0f19eed1-f6a2-4123-a9be-e5688fa7ba98","kind":"group"},{"children":[{"columnName":"description","id":"69088376-4f58-4dd7-b8ea-15049466c033","kind":"field","operator":"not empty","value":null}],"conjunction":"and","id":"62bed64f-9b90-4411-8837-8376fa3df51a","kind":"group"}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(true AND true AND (e.config->>'description' IS NOT NULL))`},
		{`{"children":[{"columnName":"numTrials","id":"b29b9844-c3c0-41de-936d-a793a30c2149","kind":"field","operator":">","value":0},{"columnName":"id","id":"9a5308c0-51e7-426e-a310-8bb67ee7d162","kind":"field","operator":"!=","value":0},{"columnName":"forkedFrom","id":"7fb4c989-983f-4cfb-8b61-59a93c946e7e","kind":"field","operator":"!=","value":1}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) > 0 AND e.id != 0 AND e.parent_id != 1)`},
		{`{"children":[{"columnName":"resourcePool","kind":"field","operator":"contains","value":"default"}],"conjunction":"and","kind":"group"}`, `(e.config->'resources'->>'resource_pool' LIKE '%default%')`},
		{`{"children":[{"columnName":"id","kind":"field","operator":"=","value":1}],"conjunction":"and","kind":"group"}`, `(e.id = 1)`},
		{`{"children":[{"columnName":"projectId","location":"LOCATION_TYPE_EXPERIMENT", "kind":"field","operator":">=","value":-1}],"conjunction":"and","kind":"group"}`, `(project_id >= -1)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":">=","value":0}],"conjunction":"and","kind":"group"}`, `((e.validation_metrics->>'validation_accuracy')::float8 >= 0)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error","kind":"field","operator":">=","value":0}],"conjunction":"and","kind":"group"}`, `((e.validation_metrics->>'validation_error')::float8 >= 0)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.x","kind":"field","operator":"=","value": 0}],"conjunction":"and","kind":"group"}`, `((e.validation_metrics->>'x')::float8 = 0)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.loss","kind":"field","operator":"!=","value":0.004}],"conjunction":"and","kind":"group"}`, `((e.validation_metrics->>'loss')::float8 != 0.004)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":"<","value":-3}],"conjunction":"and","kind":"group"}`, `((e.validation_metrics->>'validation_accuracy')::float8 < -3)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":"<=","value":10}],"conjunction":"and","kind":"group"}`, `((e.validation_metrics->>'validation_accuracy')::float8 <= 10)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":">=","value":null}],"conjunction":"and","kind":"group"}`, `(true)`},
		{`{"children":[{"columnName":"projectId","kind":"field","operator":">=","value":null}],"conjunction":"and","kind":"group"}`, `(true)`},
		{`{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"children":[{"columnName":"id","kind":"field","operator":"=","value":2},{"columnName":"id","kind":"field","operator":"=","value":3}],"conjunction":"and","kind":"group"},{"columnName":"id","id":"f2d30c06-0286-43a0-b608-d84bdf9db84d","kind":"field","operator":"=","value":4},{"children":[{"columnName":"id","id":"e55bfdc0-e775-4776-9a10-f1b9d7ce3b89","kind":"field","operator":"=","value":5}],"conjunction":"and","id":"11b13b42-15c5-495c-982d-f663187afeaf","kind":"group"}],"conjunction":"and","kind":"group"}`, `(e.id = 1 AND (e.id = 2 AND e.id = 3) AND e.id = 4 AND (e.id = 5))`},
		{`		{"children":[{"children":[{"columnName":"checkpointCount","id":"5622dd2e-3bc9-4810-a08c-c7a407b7e50c","kind":"field","operator":"=","value":4},{"columnName":"numTrials","id":"d36a7a6c-2fbf-4824-8f04-5abe0f01a71e","kind":"field","operator":"=","value":1},{"columnName":"progress","id":"71485ce5-ec86-4209-9ebd-1de883a6022a","kind":"field","operator":"=","value":100}],"conjunction":"and","id":"d3c1a590-a01a-44da-a9e1-3082034b30d0","kind":"group"}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((checkpoint_count = 4 AND (SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) = 1 AND COALESCE(progress, 0) = 100))`},
		{
			`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"=","value":32}],"conjunction":"and","kind":"group"}`,
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 = 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 = 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 = 32)
				ELSE false
			 END))`,
		},
		{
			`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":">=","value":32}],"conjunction":"and","kind":"group"}`,
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 >= 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 >= 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 >= 32)
				ELSE false
			 END))`,
		},
		{
			`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"!=","value":32}],"conjunction":"and","kind":"group"}`,
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 != 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 != 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 != 32)
				ELSE false
			 END))`,
		},
		{`{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is","value":"efficientdet_d0"}],"conjunction":"and","kind":"group"}`, `((CASE WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' = 'efficientdet_d0' ELSE false END))`},
	}
	for _, c := range validTestCases {
		var experimentFilter ExperimentFilter
		err := json.Unmarshal([]byte(c[0]), &experimentFilter)
		require.NoError(t, err)
		filterSql, err := experimentFilter.toSql()
		require.NoError(t, err)
		require.Equal(t, filterSql, c[1])
	}
}
