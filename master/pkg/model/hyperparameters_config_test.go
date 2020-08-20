package model

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
)

func TestValidateGlobalBatchSize(t *testing.T) {
	type testCase struct {
		name         string
		config       string
		errorMessage string
	}
	testCases := []testCase{
		{
			"valid hyperparameters",
			`{
			"hyperparameters": {
				"global_batch_size": {
					"type": "const",
					"val": 32
				}
			}}`,
			"",
		},
		{
			"missing global_batch_size",
			`{"hyperparameters": {}}`,
			"global_batch_size hyperparameter must be specified",
		},
		{
			"invalid global_batch_size",
			`{
			"hyperparameters": {
				"global_batch_size": {
					"type": "const",
					"val": "okok"
				}
			}}`,
			"global_batch_size must be a numeric value",
		},
		{
			"valid categorical global_batch_size",
			`{
			"hyperparameters": {
				"global_batch_size": {
				  "type": "categorical",
				  "vals": [32, 64]
				}
			}}`,
			"",
		},
		{
			"invalid categorical global_batch_size",
			`{
			"hyperparameters": {
				"global_batch_size": {
				  "type": "categorical",
				  "vals": ["32", "hello"]
				}
			}}`,
			"global_batch_size must be a numeric value",
		},
	}

	runTestCase := func(t *testing.T, tc testCase) {
		t.Run(tc.name, func(t *testing.T) {
			conf := DefaultExperimentConfig()
			assert.NilError(t, json.Unmarshal([]byte(tc.config), &conf))
			if tc.errorMessage != "" {
				assert.ErrorContains(t, check.Validate(conf.Hyperparameters), tc.errorMessage)
			} else {
				assert.NilError(t, check.Validate(conf.Hyperparameters))
			}
		})
	}

	for _, tc := range testCases {
		runTestCase(t, tc)
	}
}
