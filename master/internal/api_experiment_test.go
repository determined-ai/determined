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

		//  Invalid experiment field
		`{"children":[{"location":"LOCATION_TYPE_EXPERIMENT","columnName":"notValid","kind":"field","value":"default"}],"conjunction":"and","kind":"group"}`,
	}
	for _, c := range invalidTestCases {
		var ef experimentFilter
		err := json.Unmarshal([]byte(c), &ef)
		require.NoError(t, err)
		_, err = ef.toSQL()
		require.Error(t, err)
	}
	validTestCases := [][2]string{
		{`{"filterGroup":{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":true}`, `(true OR true OR true)`},                                                                                                                             //nolint: lll
		{`{"filterGroup":{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `e.archived = false AND (true OR true OR true)`},                                                                                                     //nolint: lll
		{`{"filterGroup":{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[{"columnName":"description","kind":"field","operator":"not empty","value":null}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(true AND true AND (e.config->>'description' IS NOT NULL))`},         //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"numTrials","kind":"field","operator":">","value":0},{"columnName":"id","kind":"field","operator":"!=","value":0},{"columnName":"forkedFrom","kind":"field","operator":"!=","value":1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) > 0 AND e.id != 0 AND e.parent_id != 1)`}, //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"description","id":"fc9571f7-d6a6-4c5a-a2f0-1f2c1df1a3ae","kind":"field","operator":"contains","value":"t\\set"}],"conjunction":"and","id":"ROOT","kind":"group"},"showArchived":false}`, `e.archived = false AND (e.config->>'description' LIKE '%t\set%')`},                                                                                               //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"description","id":"fc9571f7-d6a6-4c5a-a2f0-1f2c1df1a3ae","kind":"field","operator":"contains","value":"t\"set"}],"conjunction":"and","id":"ROOT","kind":"group"},"showArchived":false}`, `e.archived = false AND (e.config->>'description' LIKE '%t"set%')`},                                                                                               //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"contains","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(e.config->'resources'->>'resource_pool' LIKE '%default%')`},                                                                                                                                                            //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(e.id = 1)`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"tags","kind":"field","operator":"contains", "value":"val"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(e.config->>'labels' LIKE '%val%')`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"tags","kind":"field","operator":"does not contain", "value":"val"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(e.config->>'labels' NOT LIKE '%val%')`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"duration","kind":"field","operator":">", "value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(extract(seconds FROM coalesce(e.end_time, now()) - e.start_time) > 0)`},                                                                                                                                                                                                                                                                       //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"projectId","location":"LOCATION_TYPE_EXPERIMENT", "kind":"field","operator":">=","value":-1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(project_id >= -1)`},                                                                                                                                                                                                                                                                                                                                                     //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":">=","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_accuracy')::float8 >= 0)`},                                                                                                                                                                                                                                                         //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string","kind":"field","operator":"=","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(e.validation_metrics->>'validation_string' = 'string')`},                                                                                                                                                                                                                                                             //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string","kind":"field","operator":"!=","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(e.validation_metrics->>'validation_string' != 'string')`},                                                                                                                                                                                                                                                           //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string","kind":"field","operator":"contains","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(e.validation_metrics->>'validation_string' LIKE '%string%')`},                                                                                                                                                                                                                                                 //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string","kind":"field","operator":"does not contain","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(e.validation_metrics->>'validation_string' NOT LIKE '%string%')`},                                                                                                                                                                                                                                     //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error","kind":"field","operator":">=","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_error')::float8 >= 0)`},                                                                                                                                                                                                                                                               //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error","kind":"field","operator":"not empty","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_error')::float8 IS NOT NULL)`},                                                                                                                                                                                                                                                 //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error","kind":"field","operator":"is empty","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_error')::float8 IS NULL)`},                                                                                                                                                                                                                                                      //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.x","kind":"field","operator":"=","value": 0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'x')::float8 = 0)`},                                                                                                                                                                                                                                                                                              //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.loss","kind":"field","operator":"!=","value":0.004}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'loss')::float8 != 0.004)`},                                                                                                                                                                                                                                                                               //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":"<","value":-3}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_accuracy')::float8 < -3)`},                                                                                                                                                                                                                                                         //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":"<=","value":10}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_accuracy')::float8 <= 10)`},                                                                                                                                                                                                                                                       //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"projectId","kind":"field","operator":">=","value":null}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(true)`},                                                                                                                                                                                                                                                                                                                                                                                                      //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"children":[{"columnName":"id","kind":"field","operator":"=","value":2},{"columnName":"id","kind":"field","operator":"=","value":3}],"conjunction":"and","kind":"group"},{"columnName":"id","kind":"field","operator":"=","value":4},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(e.id = 1 AND (e.id = 2 AND e.id = 3) AND e.id = 4 AND (e.id = 5))`}, //nolint: lll
		{`{"filterGroup":{"children":[{"children":[{"columnName":"checkpointCount","kind":"field","operator":"=","value":4},{"columnName":"numTrials","kind":"field","operator":"=","value":1},{"columnName":"progress","kind":"field","operator":"=","value":100}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((checkpoint_count = 4 AND (SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) = 1 AND COALESCE(progress, 0) = 100))`},                                                                                    //nolint: lll
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"=","value":32}],"conjunction":"and","kind":"group"},"showArchived":true}`, //nolint: lll
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 = 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 = 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 = 32)
				ELSE false
			 END))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":">=","value":32}],"conjunction":"and","kind":"group"},"showArchived":true}`, //nolint: lll
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 >= 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 >= 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 >= 32)
				ELSE false
			 END))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"!=","value":32}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 != 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 != 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 != 32)
				ELSE false
			 END))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"is empty"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 IS NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'categorical' THEN config->'hyperparameters'->'global_batch_size'->>'vals' IS NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'global_batch_size') IS NULL
				ELSE false
			 END)
			)`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"not empty"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 IS NOT NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'categorical' THEN config->'hyperparameters'->'global_batch_size'->>'vals' IS NOT NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'global_batch_size') IS NOT NULL
				ELSE false
			 END)
			)`,
		},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is","value":"efficientdet_d0"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((CASE WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' = 'efficientdet_d0' ELSE false END))`},                         //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is","value":"efficientdet_d0"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `e.archived = false AND ((CASE WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' = 'efficientdet_d0' ELSE false END))`}, //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is empty"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((CASE
				WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' IS NULL
				WHEN config->'hyperparameters'->'model'->>'type' = 'categorical' THEN config->'hyperparameters'->'model'->>'vals' IS NULL
				ELSE false
			 END)
			)`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad","kind":"field","operator":"contains", "value":8}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((CASE
					WHEN config->'hyperparameters'->'clip_grad'->>'type' = 'categorical' THEN (config->'hyperparameters'->'clip_grad'->>'vals')::jsonb ? '8'
					WHEN config->'hyperparameters'->'clip_grad'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'clip_grad'->>'minval')::float8 <= 8 OR (config->'hyperparameters'->'clip_grad'->>'maxval')::float8 >= 8
					ELSE false
				 END))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad","kind":"field","operator":"does not contain", "value":8}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((CASE
					WHEN config->'hyperparameters'->'clip_grad'->>'type' = 'categorical' THEN ((config->'hyperparameters'->'clip_grad'->>'vals')::jsonb ? '8') IS NOT TRUE
					WHEN config->'hyperparameters'->'clip_grad'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'clip_grad'->>'minval')::float8 >= 8 OR (config->'hyperparameters'->'clip_grad'->>'maxval')::float8 <= 8
					ELSE false
				 END))`},
	}
	for _, c := range validTestCases {
		var efr experimentFilterRoot
		err := json.Unmarshal([]byte(c[0]), &efr)
		require.NoError(t, err)
		filterSQL, err := efr.toSQL()
		require.NoError(t, err)
		require.Equal(t, filterSQL, c[1])
	}
}
