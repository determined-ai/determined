//go:build integration
// +build integration

package internal

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
)

func TestExperimentSearchApiFilterParsing(t *testing.T) {
	setupAPITest(t, nil)
	invalidTestCases := []string{
		// No operator specified in field
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":false}`,

		// No conjunction in group
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"=","value":"default"}],"kind":"group"},"showArchived":false}`,

		// invalid group conjunction
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"=","value":"default"}],"conjunction":"invalid","kind":"group"},"showArchived":false}`,

		// invalid operator
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"invalid","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":false}`,

		//  Invalid experiment field
		`{"filterGroup":{"children":[{"location":"LOCATION_TYPE_EXPERIMENT","columnName":"notValid","kind":"field","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":false}`,
	}
	for _, c := range invalidTestCases {
		q := db.Bun().NewSelect()
		var efr experimentFilterRoot
		err := json.Unmarshal([]byte(c), &efr)
		require.NoError(t, err)
		_, err = efr.toSQL(q)
		require.Error(t, err)
	}
	validTestCases := [][2]string{
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1}],"conjunction":"and","kind":"group"},"showArchived":false}`, `((e.id = 1)) AND (e.archived = false)`},
		{`{"filterGroup":{"children":[{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":3},{"columnName":"id","kind":"field","operator":"=","value":4}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5},{"columnName":"id","kind":"field","operator":"=","value":6}],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `(((e.id = 1)) AND ((e.id = 2))) OR (((e.id = 3)) AND ((e.id = 4))) OR (((e.id = 5)) AND ((e.id = 6))) AND (e.archived = false)`}, //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2}],"conjunction":"and","kind":"group"},"showArchived":false}`, `((e.id = 1)) AND ((e.id = 2)) AND (e.archived = false)`},
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2}],"conjunction":"or","kind":"group"},"showArchived":false}`, `((e.id = 1)) OR ((e.id = 2)) AND (e.archived = false)`},
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2},{"children":[{"columnName":"id","kind":"field","operator":"=","value":3},{"columnName":"id","kind":"field","operator":"=","value":4}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5},{"children":[{"columnName":"id","kind":"field","operator":"=","value":6},{"columnName":"id","kind":"field","operator":"=","value":7}],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `((e.id = 1)) OR ((e.id = 2)) OR (((e.id = 3)) AND ((e.id = 4))) OR (((e.id = 5)) OR (((e.id = 6)) AND ((e.id = 7)))) AND (e.archived = false)`},
		{`{"filterGroup":{"children":[{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"columnName":"id","kind":"field","operator":"=","value":2}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":3},{"columnName":"id","kind":"field","operator":"=","value":4}],"conjunction":"and","kind":"group"},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5},{"columnName":"id","kind":"field","operator":"=","value":6}],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `(((e.id = 1)) AND ((e.id = 2))) OR (((e.id = 3)) AND ((e.id = 4))) OR (((e.id = 5)) AND ((e.id = 6))) AND (e.archived = false)`}, //nolint: lll
		{`{"filterGroup":{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `((true)) OR ((true)) OR ((true)) AND (e.archived = false)`},                                                                                                   //nolint: lll
		{`{"filterGroup":{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[{"columnName":"description","kind":"field","operator":"not empty","value":null}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((true)) AND ((true)) AND (((e.config->>'description' IS NOT NULL)))`},         //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"numTrials","kind":"field","operator":">","value":0},{"columnName":"id","kind":"field","operator":"!=","value":0},{"columnName":"forkedFrom","kind":"field","operator":"!=","value":1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) > 0)) AND ((e.id != 0)) AND ((e.parent_id != 1))`}, //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"description","kind":"field","operator":"contains","value":"t\\set"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `((e.config->>'description' LIKE '%t\set%')) AND (e.archived = false)`},                                                                                                                                                             //nolint: lll                                                                                                                                                        //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"contains","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.config->'resources'->>'resource_pool' LIKE '%default%'))`},                                                                                                                                                                    //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.id = 1))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_DATE","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"startTime","kind":"field","operator":">", "value":"2021-04-14T14:14:18.915483952Z"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `((e.start_time > '2021-04-14T14:14:18.915483952Z')) AND (e.archived = false)`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_DATE","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"endTime","kind":"field","operator":"<=", "value":"2021-04-14T14:14:18.915483952Z"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `((e.end_time <= '2021-04-14T14:14:18.915483952Z')) AND (e.archived = false)`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"tags","kind":"field","operator":"contains", "value":"val"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.config->>'labels' LIKE '%val%'))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"tags","kind":"field","operator":"does not contain", "value":"val"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.config->>'labels' NOT LIKE '%val%'))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"duration","kind":"field","operator":">", "value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((extract(seconds FROM coalesce(e.end_time, now()) - e.start_time) > 0))`},                                                                                                                                                                                                                                                                                       //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"projectId","location":"LOCATION_TYPE_EXPERIMENT", "kind":"field","operator":">=","value":-1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((project_id >= -1))`},                                                                                                                                                                                                                                                                                                                                                                     //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":">=","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.validation_metrics->>'validation_accuracy')::float8 >= 0))`},                                                                                                                                                                                                                                                                         //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string","kind":"field","operator":"=","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_string' = 'string'))`},                                                                                                                                                                                                                                                                             //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string","kind":"field","operator":"!=","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_string' != 'string'))`},                                                                                                                                                                                                                                                                           //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string","kind":"field","operator":"contains","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_string' LIKE '%string%'))`},                                                                                                                                                                                                                                                                 //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_string","kind":"field","operator":"does not contain","value":"string"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.validation_metrics->>'validation_string' NOT LIKE '%string%'))`},                                                                                                                                                                                                                                                     //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error","kind":"field","operator":">=","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.validation_metrics->>'validation_error')::float8 >= 0))`},                                                                                                                                                                                                                                                                               //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error","kind":"field","operator":"not empty","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.validation_metrics->>'validation_error')::float8 IS NOT NULL))`},                                                                                                                                                                                                                                                                 //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error","kind":"field","operator":"is empty","value":0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.validation_metrics->>'validation_error')::float8 IS NULL))`},                                                                                                                                                                                                                                                                      //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.x","kind":"field","operator":"=","value": 0}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.validation_metrics->>'x')::float8 = 0))`},                                                                                                                                                                                                                                                                                                              //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.loss","kind":"field","operator":"!=","value":0.004}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.validation_metrics->>'loss')::float8 != 0.004))`},                                                                                                                                                                                                                                                                                               //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":"<","value":-3}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.validation_metrics->>'validation_accuracy')::float8 < -3))`},                                                                                                                                                                                                                                                                         //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","kind":"field","operator":"<=","value":10}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((e.validation_metrics->>'validation_accuracy')::float8 <= 10))`},                                                                                                                                                                                                                                                                       //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"projectId","kind":"field","operator":">=","value":null}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((true))`},                                                                                                                                                                                                                                                                                                                                                                                                                      //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1},{"children":[{"columnName":"id","kind":"field","operator":"=","value":2},{"columnName":"id","kind":"field","operator":"=","value":3}],"conjunction":"and","kind":"group"},{"columnName":"id","kind":"field","operator":"=","value":4},{"children":[{"columnName":"id","kind":"field","operator":"=","value":5}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.id = 1)) AND (((e.id = 2)) AND ((e.id = 3))) AND ((e.id = 4)) AND (((e.id = 5)))`}, //nolint: lll
		{`{"filterGroup":{"children":[{"children":[{"columnName":"checkpointCount","kind":"field","operator":"=","value":4},{"columnName":"numTrials","kind":"field","operator":"=","value":1},{"columnName":"progress","kind":"field","operator":"=","value":100}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((checkpoint_count = 4)) AND (((SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) = 1)) AND ((COALESCE(progress, 0) = 100)))`},                                                                                            //nolint: lll
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"=","value":32}],"conjunction":"and","kind":"group"},"showArchived":true}`, //nolint: lll
			`(((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 = 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 = 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 = 32)
				ELSE false
			 END)))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":">=","value":32}],"conjunction":"and","kind":"group"},"showArchived":true}`, //nolint: lll
			`(((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 >= 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 >= 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 >= 32)
				ELSE false
			 END)))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_DATE","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_start","kind":"field","operator":">=","value":"2021-04-14T14:14:18.915483952Z"}],"conjunction":"and","kind":"group"},"showArchived":true}`, //nolint: lll
			`(((CASE
				WHEN config->'hyperparameters'->'global_batch_start'->>'type' = 'const' THEN config->'hyperparameters'->'global_batch_start'->>'val' >= '2021-04-14T14:14:18.915483952Z'
				ELSE false
			 END)))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"!=","value":32}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`(((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 != 32
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN ((config->'hyperparameters'->'global_batch_size'->>'minval')::float8 != 32 OR (config->'hyperparameters'->'global_batch_size'->>'maxval')::float8 != 32)
				ELSE false
			 END)))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"is empty"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`(((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 IS NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'categorical' THEN config->'hyperparameters'->'global_batch_size'->>'vals' IS NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'global_batch_size') IS NULL
				ELSE false
			 END)))`,
		},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","kind":"field","operator":"not empty"}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`(((CASE
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'const' THEN (config->'hyperparameters'->'global_batch_size'->>'val')::float8 IS NOT NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' = 'categorical' THEN config->'hyperparameters'->'global_batch_size'->>'vals' IS NOT NULL
				WHEN config->'hyperparameters'->'global_batch_size'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'global_batch_size') IS NOT NULL
				ELSE false
			 END)))`,
		},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is","value":"efficientdet_d0"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((CASE WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' = 'efficientdet_d0' ELSE false END)))`},                           //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is","value":"efficientdet_d0"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `(((CASE WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' = 'efficientdet_d0' ELSE false END))) AND (e.archived = false)`}, //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is empty"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((CASE
				WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' IS NULL
				WHEN config->'hyperparameters'->'model'->>'type' = 'categorical' THEN config->'hyperparameters'->'model'->>'vals' IS NULL
				ELSE false
			 END)))`},
		{
			`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad","kind":"field","operator":"contains", "value":8}],"conjunction":"and","kind":"group"},"showArchived":true}`,
			`(((CASE
					WHEN config->'hyperparameters'->'clip_grad'->>'type' = 'categorical' THEN (config->'hyperparameters'->'clip_grad'->>'vals')::jsonb ? '8'
					WHEN config->'hyperparameters'->'clip_grad'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'clip_grad'->>'minval')::float8 <= 8 OR (config->'hyperparameters'->'clip_grad'->>'maxval')::float8 >= 8
					ELSE false
				 END)))`,
		},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad","kind":"field","operator":"does not contain", "value":8}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((CASE
					WHEN config->'hyperparameters'->'clip_grad'->>'type' = 'categorical' THEN ((config->'hyperparameters'->'clip_grad'->>'vals')::jsonb ? '8') IS NOT TRUE
					WHEN config->'hyperparameters'->'clip_grad'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'clip_grad'->>'minval')::float8 >= 8 OR (config->'hyperparameters'->'clip_grad'->>'maxval')::float8 <= 8
					ELSE false
				 END)))`},
	}
	for _, c := range validTestCases {
		q := db.Bun().NewSelect()
		var efr experimentFilterRoot
		err := json.Unmarshal([]byte(c[0]), &efr)
		require.NoError(t, err)
		_, err = efr.toSQL(q)
		require.NoError(t, err)
		require.Equal(t, q.String(), fmt.Sprintf(`SELECT * WHERE %v`, c[1]))
	}
}
