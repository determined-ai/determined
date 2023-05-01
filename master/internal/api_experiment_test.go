//go:build integration
// +build integration

package internal

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/stretchr/testify/require"
)

func TestExperimentSearchApiFilterParsing(t *testing.T) {
	setupAPITest(t, nil)
	invalidTestCases := []string{
		// No operator specified in field
		//`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":false}`,

		// No conjunction in group
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"=","value":"default"}],"kind":"group"},"showArchived":false}`,

		// invalid group conjunction
		`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"=","value":"default"}],"conjunction":"invalid","kind":"group"},"showArchived":false}`,

		// invalid operator
		//`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"invalid","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":false}`,

		//  Invalid experiment field
		//`{"filterGroup":{"children":[{"location":"LOCATION_TYPE_EXPERIMENT","columnName":"notValid","kind":"field","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":false}`,
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
		{`{"filterGroup":{"children":[{"columnName":"id","id":"1fc2daa6-78b9-44c2-a333-d780be0efc71","kind":"field","operator":"=","value":1}],"conjunction":"and","id":"ROOT","kind":"group"},"showArchived":false}`, `((e.id = 1)) AND (e.archived = false))`},
		{`{"filterGroup":{"children":[{"children":[{"columnName":"id","id":"49d297c4-415f-4d65-b4ff-f3aed8f5f459","kind":"field","operator":"=","value":1},{"columnName":"id","id":"69ad72b5-0b96-4aee-82ca-f6d375503bc3","kind":"field","operator":"=","value":2}],"conjunction":"and","id":"fda5e327-394c-4572-9b18-bbe011b0ec36","kind":"group"},{"children":[{"columnName":"id","id":"88bccb4a-04aa-4d3e-8792-f8a97e0f549c","kind":"field","operator":"=","value":3},{"columnName":"id","id":"0ce27d9d-4c6a-4631-a885-eb8e7cd0dc2f","kind":"field","operator":"=","value":4}],"conjunction":"and","id":"546c2c7f-31bd-4b9d-84b1-3e4f17c7c565","kind":"group"},{"children":[{"columnName":"id","id":"8225e06f-f9c4-4d05-a156-23964eefd3e1","kind":"field","operator":"=","value":5},{"columnName":"id","id":"c06eec31-d6b3-4690-9fd3-e3526a981017","kind":"field","operator":"=","value":6}],"conjunction":"and","id":"20d450ce-d361-4965-bb3e-38d39d1a2f7f","kind":"group"}],"conjunction":"or","id":"ROOT","kind":"group"},"showArchived":false}`, `(((e.id = 1)) AND ((e.id = 2))) OR (((e.id = 3)) AND ((e.id = 4))) OR (((e.id = 5)) AND ((e.id = 6))) AND (e.archived = false))`}, //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"id","id":"1fc2daa6-78b9-44c2-a333-d780be0efc71","kind":"field","operator":"=","value":1},{"columnName":"id","id":"b01a54e4-26c2-46a7-b7ea-9e906829c240","kind":"field","operator":"=","value":2}],"conjunction":"and","id":"ROOT","kind":"group"},"showArchived":false}`, `((e.id = 1)) AND ((e.id = 2)) AND (e.archived = false))`},
		{`{"filterGroup":{"children":[{"columnName":"id","id":"1fc2daa6-78b9-44c2-a333-d780be0efc71","kind":"field","operator":"=","value":1},{"columnName":"id","id":"b01a54e4-26c2-46a7-b7ea-9e906829c240","kind":"field","operator":"=","value":2}],"conjunction":"or","id":"ROOT","kind":"group"},"showArchived":false}`, `((e.id = 1)) OR ((e.id = 2)) AND (e.archived = false))`},
		{`{"filterGroup":{"children":[{"columnName":"id","id":"1fc2daa6-78b9-44c2-a333-d780be0efc71","kind":"field","operator":"=","value":1},{"columnName":"id","id":"b01a54e4-26c2-46a7-b7ea-9e906829c240","kind":"field","operator":"=","value":2},{"children":[{"columnName":"id","id":"f0748753-f020-4937-bb0b-f545c0ee5561","kind":"field","operator":"=","value":3},{"columnName":"id","id":"f4e544bc-76ae-4b6f-a9b7-24f3ca3570a2","kind":"field","operator":"=","value":4}],"conjunction":"and","id":"5f2bc29f-c41d-480f-83b4-b4c5534b7a42","kind":"group"},{"children":[{"columnName":"id","id":"5453586a-043e-4bca-8a27-36127bc983ea","kind":"field","operator":"=","value":5},{"children":[{"columnName":"id","id":"eef8f7c6-9141-45bf-8ea0-0665345ecfd3","kind":"field","operator":"=","value":6},{"columnName":"id","id":"58973dae-9684-490d-ac44-e1205ce8e433","kind":"field","operator":"=","value":7}],"conjunction":"and","id":"dc4a1df6-8999-401f-bd1f-16392bb92f63","kind":"group"}],"conjunction":"or","id":"244fd8c9-4284-48e5-8a10-0fbb597f9e5e","kind":"group"}],"conjunction":"or","id":"ROOT","kind":"group"},"showArchived":false}`, `((e.id = 1)) OR ((e.id = 2)) OR (((e.id = 3)) AND ((e.id = 4))) OR (((e.id = 5)) OR (((e.id = 6)) AND ((e.id = 7)))) AND (e.archived = false))`},
		{`{"filterGroup":{"children":[{"children":[{"columnName":"id","id":"a4c7ab0b-4c2f-4a61-a5c8-103b183afd00","kind":"field","operator":"=","value":1},{"columnName":"id","id":"88b65b34-b9b2-4f57-8984-1fa7afed9756","kind":"field","operator":"=","value":2}],"conjunction":"and","id":"c2cb512a-efff-4b96-87de-2de6c4b0c06b","kind":"group"},{"children":[{"columnName":"id","id":"21f89291-ee8e-46b5-a13b-9c8af8a94d6d","kind":"field","operator":"=","value":3},{"columnName":"id","id":"55a3db9e-2fb7-4c72-9639-69946b5fa9fc","kind":"field","operator":"=","value":4}],"conjunction":"and","id":"459b6a90-f76f-4733-becb-cbc68f763a96","kind":"group"},{"children":[{"columnName":"id","id":"564ba579-5531-43ed-83e8-d39716cda285","kind":"field","operator":"=","value":5},{"columnName":"id","id":"1b812afd-1602-4826-a176-36013867e762","kind":"field","operator":"=","value":6}],"conjunction":"and","id":"de10283e-7d70-462b-a330-c70b7c88bccf","kind":"group"}],"conjunction":"or","id":"ROOT","kind":"group"},"showArchived":false}`, `(((e.id = 1)) AND ((e.id = 2))) OR (((e.id = 3)) AND ((e.id = 4))) OR (((e.id = 5)) AND ((e.id = 6))) AND (e.archived = false))`}, //nolint: lll
		{`{"filterGroup":{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"}],"conjunction":"or","kind":"group"},"showArchived":false}`, `((true)) OR ((true)) OR ((true)) AND (e.archived = false))`},                                                                                                  //nolint: lll
		{`{"filterGroup":{"children":[{"children":[],"conjunction":"and","kind":"group"},{"children":[],"conjunction":"and","kind":"group"},{"children":[{"columnName":"description","kind":"field","operator":"not empty","value":null}],"conjunction":"and","kind":"group"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((true)) AND ((true)) AND (((e.config->>'description' IS NOT NULL)))`},         //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"numTrials","kind":"field","operator":">","value":0},{"columnName":"id","kind":"field","operator":"!=","value":0},{"columnName":"forkedFrom","kind":"field","operator":"!=","value":1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id) > 0)) AND ((e.id != 0)) AND ((e.parent_id != 1))`}, //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"description","kind":"field","operator":"contains","value":"t\\set"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `((e.config->>'description' LIKE '%t\set%')) AND (e.archived = false))`},                                                                                                                                                            //nolint: lll                                                                                                                                                        //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"resourcePool","kind":"field","operator":"contains","value":"default"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.config->'resources'->>'resource_pool' LIKE '%default%'))`},                                                                                                                                                                    //nolint: lll
		{`{"filterGroup":{"children":[{"columnName":"id","kind":"field","operator":"=","value":1}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.id = 1))`},
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_DATE","location":"LOCATION_TYPE_EXPERIMENT", "columnName":"startTime","kind":"field","operator":">", "value":"2021-04-14T14:14:18.915483952Z"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `((e.start_time > '2021-04-14T14:14:18.915483952Z'))`},
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
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is","value":"efficientdet_d0"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((CASE WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' = 'efficientdet_d0' ELSE false END)))`},                            //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is","value":"efficientdet_d0"}],"conjunction":"and","kind":"group"},"showArchived":false}`, `(((CASE WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' = 'efficientdet_d0' ELSE false END))) AND (e.archived = false))`}, //nolint: lll
		{`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_TEXT","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.model","kind":"field","operator":"is empty"}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((CASE
				WHEN config->'hyperparameters'->'model'->>'type' = 'const' THEN config->'hyperparameters'->'model'->>'val' IS NULL
				WHEN config->'hyperparameters'->'model'->>'type' = 'categorical' THEN config->'hyperparameters'->'model'->>'vals' IS NULL
				ELSE false
			 END)))`},
		// {`{"filterGroup":{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.clip_grad","kind":"field","operator":"contains", "value":8}],"conjunction":"and","kind":"group"},"showArchived":true}`, `(((CASE
		//  			WHEN config->'hyperparameters'->'clip_grad'->>'type' = 'categorical' THEN (config->'hyperparameters'->'clip_grad'->>'vals')::jsonb ? '8'
		//  			WHEN config->'hyperparameters'->'clip_grad'->>'type' IN ('int', 'double', 'log') THEN (config->'hyperparameters'->'clip_grad'->>'minval')::float8 <= 8 OR (config->'hyperparameters'->'clip_grad'->>'maxval')::float8 >= 8
		//  			ELSE false
		//  		 END)))`},
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
