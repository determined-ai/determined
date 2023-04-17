package internal

import (
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
	}
	for _, c := range invalidTestCases {
		_, err := parseFilter(c)
		require.Error(t, err)
	}
	validTestCases := [][2]string{
		{`experiment.user:"username"`, `COALESCE(u.display_name, u.username) = 'username'`},
		{"-experiment.projectId:123456789", "project_id != 123456789"},
		{"experiment.checkpointSize<=12", "checkpoint_size<=12"},
		{"experiment.numTrials>=9.22",
			"(SELECT COUNT(*) FROM trials t WHERE e.id = t.experiment_id)>=9.22"},
		{
			"validation.validation_accuracy>1",
			"(e.validation_metrics->>'validation_accuracy')::float8>1",
		},
		{
			"validation.validation_loss<-1",
			"(e.validation_metrics->>'validation_loss')::float8<-1",
		},
		{
			"validation.validation_test>-10.98",
			"(e.validation_metrics->>'validation_test')::float8>-10.98",
		},
		{
			"hp.global_batch_size>=32",
			"(e.config->'hyperparameters'->'global_batch_size'->>'val')::float8>=32",
		},
		{
			"hp.global_batch_size<=-64",
			"(e.config->'hyperparameters'->'global_batch_size'->>'val')::float8<=-64",
		},
		{`experiment.checkpointCount:null`, `checkpoint_count IS NULL`},
		{`-experiment.endTime:null`, `e.end_time IS NOT NULL`},
		{`experiment.description~"like"`, `e.config->>'description' LIKE '%like%'`},
		{`(experiment.description~"like" AND -experiment.description~"notlike")`,
			`(e.config->>'description' LIKE '%like%' AND e.config->>'description' NOT LIKE '%notlike%')`}, //nolint: lll
		{`experiment.startTime>="2023-01-06T19:06:25.053893089Z" OR experiment.endTime<="2023-01-06T19:08:33.219618082Z"`, //nolint: lll
			`e.start_time>='2023-01-06T19:06:25.053893089Z' OR e.end_time<='2023-01-06T19:08:33.219618082Z'`}, //nolint: lll
		{`(experiment.description~"experiment description" AND (-experiment.id:456 OR -experiment.resourcePool~"test\"s comma value\"s"))`, //nolint: lll
			`(e.config->>'description' LIKE '%experiment description%' AND (e.id != 456 OR e.config->'resources'->>'resource_pool' NOT LIKE '%test\"s comma value\"s%'))`}, //nolint: lll
		{
			`(experiment.forkedFrom:5 OR (-validation.error:1 AND hp.hyperparameter<=10))`,
			`(e.parent_id = 5 OR ((e.validation_metrics->>'error')::float8 != 1 AND (e.config->'hyperparameters'->'hyperparameter'->>'val')::float8<=10))`, //nolint: lll
		}, //nolint: lll
	}
	for _, c := range validTestCases {
		result, err := parseFilter(c[0])
		require.NoError(t, err)
		require.Equal(t, c[1], *result)
	}
}
