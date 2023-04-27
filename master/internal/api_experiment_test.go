package internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExperimentSearchApiFilterParsing(t *testing.T) {
	invalidTestCases := []string{
		"(",
		")",
		"())junk",
		")morejunk()",
		"(((junk(((()",
		")((((otherjunk)((()))))))",
		"",
		"        ",
		"   ()     ",
		"  (    )  ",
		"()",
		"()()",
		"()       ()",
		"(()())",
		`string:"value"`,
		"-number:123456789",
		"anumber<=12.34",
		"thisnumber>=9.22",
		"string:null",
		"-value:null",
		`str~"like"`,
		`str~"like" AND -otherstr~"notlike")`,
		`experiment.user  :   "username"`,
		"-experiment.projectId: 123456789",
		"experiment.checkpointSize <=12",
		"experiment.checkpointSize <12",
		"experiment.checkpointSize <=  12",
		`experiment.user:   "username"`,
		"hp.global_batch_size <=-64",
		"validation.validation_test>   -10.98",
		"validation.validation_test   >   -10.98",
		"hp.global_batch_size<=-90 AND hp.global_batch_size>= -64",
		"hp.global_batch_size<=-64 AND hp.global_batch_size <=-64",
		"hp.global_batch_size<=-64 OR validation.validation_test   >   -10.98",
		"hp.global_batch_size:-64 OR validation.validation_test< 20",
		"hp.global_batch_size: 64 OR validation.validation_test<20",
		`hp.global_batch_size:"string value" OR hp.global_batch_size<20`,
		`hp.global_batch_size:"20" OR hp.global_batch_size<20`,
	}
	for _, c := range invalidTestCases {
		_, err := parseFilter(c)
		require.Error(t, err)
	}
	validTestCases := [][2]string{
		{`{"children":[{"columnName":"resourcePool","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"contains","value":"default"}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(e.config->'resources'->>'resource_pool' LIKE '%default%')`},
		{`{"children":[{"columnName":"id","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"=","value":1}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(e.id = 1)`},
		{`{"children":[{"columnName":"projectId","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":">=","value":-1}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(project_id >= -1)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":">=","value":0}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'validation_accuracy')::float8 >= 0)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_error","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":">=","value":0}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'validation_error')::float8 >= 0)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.x","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"=","value": 0}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'x')::float8 = 0)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.loss","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"!=","value":0.004}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'loss')::float8 != 0.004)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"<","value":-3}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'validation_accuracy')::float8 < -3)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_VALIDATIONS", "columnName":"validation.validation_accuracy","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"<=","value":10}],"conjunction":"and","id":"ROOT","kind":"group"}`, `((e.validation_metrics->>'validation_accuracy')::float8 <= 10)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.global_batch_size","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"=","value":32}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(("e.config->'hyperparameters'->'global_batch_size'->>'val'")::float8 = 32)`},
		{`{"children":[{"type":"COLUMN_TYPE_NUMBER","location":"LOCATION_TYPE_HYPERPARAMETERS", "columnName":"hp.nested.hp.global_batch_size","id":"10043dda-2187-45d4-92ce-b9ade5244b6f","kind":"field","operator":"!=","value":32}],"conjunction":"and","id":"ROOT","kind":"group"}`, `(("e.config->'hyperparameters'->'nested'->'hp'->'global_batch_size'->>'val'")::float8 != 32)`},
		// {"-experiment.projectId:123456789", "project_id != 123456789"},
		// {"experiment.checkpointSize<=12", "checkpoint_size<=12"},
		// {
		// 	"experiment.numTrials>=9.22",
		// 	"(SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id)>=9.22",
		// },
		// {
		// 	"validation.validation_accuracy>1",
		// 	"(e.validation_metrics->>'validation_accuracy')::float8>1",
		// },
		// {
		// 	"validation.validation_loss<-1",
		// 	"(e.validation_metrics->>'validation_loss')::float8<-1",
		// },
		// {
		// 	"validation.validation_test>-10.98",
		// 	"(e.validation_metrics->>'validation_test')::float8>-10.98",
		// },
		// {
		// 	"hp.global_batch_size>=32",
		// 	"(e.config->'hyperparameters'->'global_batch_size'->>'val')::float8>=32",
		// },
		// {
		// 	"hp.global_batch_size<=-64",
		// 	"(e.config->'hyperparameters'->'global_batch_size'->>'val')::float8<=-64",
		// },
		// {
		// 	`hp.some_string:"string"`,
		// 	"e.config->'hyperparameters'->'some_string'->>'val' = 'string'",
		// },
		// {
		// 	"validation.validation_test_value:null",
		// 	"e.validation_metrics->>'validation_test_value' IS NULL",
		// },
		// {`experiment.checkpointCount:null`, `checkpoint_count IS NULL`},
		// {`-experiment.endTime:null`, `e.end_time IS NOT NULL`},
		// {`experiment.description~"like"`, `e.config->>'description' LIKE '%like%'`},
		// {
		// 	`(experiment.description~"like" AND -experiment.description~"notlike")`,
		// 	`(e.config->>'description' LIKE '%like%' AND e.config->>'description' NOT LIKE '%notlike%')`,
		// }, //nolint: lll
		// {`experiment.startTime>="2023-01-06T19:06:25.053893089Z" OR experiment.endTime<="2023-01-06T19:08:33.219618082Z"`, //nolint: lll
		// 	`e.start_time>='2023-01-06T19:06:25.053893089Z' OR e.end_time<='2023-01-06T19:08:33.219618082Z'`}, //nolint: lll
		// {`(experiment.description~"experiment description" AND (-experiment.id:456 OR -experiment.resourcePool~"test\"s comma value\"s"))`, //nolint: lll
		// 	`(e.config->>'description' LIKE '%experiment description%' AND (e.id != 456 OR e.config->'resources'->>'resource_pool' NOT LIKE '%test\"s comma value\"s%'))`}, //nolint: lll
		// {
		// 	`(experiment.forkedFrom:5 OR (-validation.error:1 AND hp.hyperparameter<=10))`,
		// 	`(e.parent_id = 5 OR ((e.validation_metrics->>'error')::float8 != 1 AND (e.config->'hyperparameters'->'hyperparameter'->>'val')::float8<=10))`, //nolint: lll
		// },
		// {
		// 	`validation.validation_test_value>="2023-01-06T19:06:25.053893089Z"`,
		// 	`e.validation_metrics->>'validation_test_value'>='2023-01-06T19:06:25.053893089Z'`,
		// },
		// {
		// 	`(-validation.error:null OR (-validation.error:1 AND hp.hyperparameter<=10))`,
		// 	`((e.validation_metrics->>'error')::float8 IS NOT NULL OR ((e.validation_metrics->>'error')::float8 != 1 AND (e.config->'hyperparameters'->'hyperparameter'->>'val')::float8<=10))`, //nolint: lll
		// },
		// {
		// 	`(validation.error:null OR (-validation.error:"1" AND hp.hyperparameter<=10))`,
		// 	`(e.validation_metrics->>'error' IS NULL OR (e.validation_metrics->>'error' != '1' AND (e.config->'hyperparameters'->'hyperparameter'->>'val')::float8<=10))`, //nolint: lll
		// },
	}
	for _, c := range validTestCases {
		var experimentFilter ExperimentFilter
		err := json.Unmarshal([]byte(c[0]), &experimentFilter)
		require.NoError(t, err)
		filterSql, err := experimentFilter.toSql()
		require.NoError(t, err)
		require.Equal(t, c[1], filterSql)
	}
}
