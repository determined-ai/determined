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
		{`{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"}`, `(true OR true OR true)`},
		{`{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[{"columnName":"description","kind":"field","operator":"not empty","value":null}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"}`, `(true AND true AND (e.config->>'description' IS NOT NULL))`},
		{`{"children":[{"columnName":"numTrials","kind":"field","operator":">","value":0},{"columnName":"id","kind":"field","operator":"!=","value":0},{"columnName":"forkedFrom","kind":"field","operator":"!=","value":1}],"conjunction":"and","kind":"group"}`, `((SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) > 0 AND e.id != 0 AND e.parent_id != 1)`},
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
		{`{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"children":[{"columnName":"id","kind":"field","operator":"=","value":2},{"columnName":"id","kind":"field","operator":"=","value":3}],"conjunction":"and","kind":"group"},{"columnName":"id","kind":"field","operator":"=","value":4},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"}`, `(e.id = 1 AND (e.id = 2 AND e.id = 3) AND e.id = 4 AND (e.id = 5))`},
		{`{"children":[{"children":[{"columnName":"checkpointCount","kind":"field","operator":"=","value":4},{"columnName":"numTrials","kind":"field","operator":"=","value":1},{"columnName":"progress","kind":"field","operator":"=","value":100}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"}`, `((checkpoint_count = 4 AND (SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) = 1 AND COALESCE(progress, 0) = 100))`},
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
