package internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExperimentSearchApiFilterParsing(t *testing.T) {
	invalidTestCases := []string{
		// TOOD add invalid test cases
		" ",
	}
	for _, c := range invalidTestCases {
		_, err := parseFilter(c)
		require.Error(t, err)
	}
	validTestCases := [][2]string{
		// TOOD add more valid test cases
		{`{"children":[{"columnName":"resourcePool","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"contains","value":"default"}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(e.config->'resources'->>'resource_pool' LIKE '%default%')`},
		{`{"children":[{"columnName":"id","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"=","value":1}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(e.id = 1)`},
		{`{"children":[{"columnName":"projectId","location":"LOCATION_TYPE_EXPERIMENT", "id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":">=","value":-1}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(project_id >= -1)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":">=","value":0}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'validation_accuracy')::float8 >= 0)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":">=","value":0}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'validation_error')::float8 >= 0)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.x","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"=","value": 0}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'x')::float8 = 0)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.loss","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"!=","value":0.004}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'loss')::float8 != 0.004)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"<","value":-3}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'validation_accuracy')::float8 < -3)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"<=","value":10}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'validation_accuracy')::float8 <= 10)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":">=","value":null}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(true)`},
		{`{"children":[{"columnName":"projectId","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":">=","value":null}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(true)`},
		{`{"children":[{"columnName":"id","id":"c6d84ec0-bcb2-4e79-ab46-d0fc94ef49ed","kind":"field","operator":"=","value":1},{"children":[{"columnName":"id","id":"975e0a2e-457a-4517-baca-3f7f488e6b92","kind":"field","operator":"=","value":2},{"columnName":"id","id":"5979523b-5ec0-4184-ace8-d5e66a2d9f3e","kind":"field","operator":"=","value":3}],"conjunction":"and","id":"ea3591bd-8481-4239-b4c7-2516e7657db7","kind":"group"},{"columnName":"id","id":"f2d30c06-0286-43a0-b608-d84bdf9db84d","kind":"field","operator":"=","value":4},{"children":[{"columnName":"id","id":"e55bfdc0-e775-4776-9a10-f1b9d7ce3b89","kind":"field","operator":"=","value":5}],"conjunction":"and","id":"11b13b42-15c5-495c-982d-f663187afeaf","kind":"group"}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(e.id = 1 AND (e.id = 2 AND e.id = 3) AND e.id = 4 AND (e.id = 5))`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"=","value":32}],"conjunction":"and","id":"ROOT","kind":"group"}`,
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 = 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 = 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 = 32)
				ELSE false
			 END))`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":">=","value":32}],"conjunction":"and","id":"ROOT","kind":"group"}`,
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 >= 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 >= 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 >= 32)
				ELSE false
			 END))`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"!=","value":32}],"conjunction":"and","id":"ROOT","kind":"group"}`,
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 != 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 != 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 != 32)
				ELSE false
			 END))`},
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
