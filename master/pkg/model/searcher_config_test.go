package model

import (
	"encoding/json"
	"fmt"
	"testing"

	"gotest.tools/assert"
)

func TestASHAMaxConcurrentTrials(t *testing.T) {
	var actual = DefaultExperimentConfig(nil).Searcher
	assert.NilError(t, json.Unmarshal([]byte(`
{
  "name": "adaptive_asha",
  "metric": "metric",
  "max_concurrent_trials": 8
}
`), &actual))
	expected := SearcherConfig{
		Metric:          "metric",
		SmallerIsBetter: true,
		AdaptiveASHAConfig: &AdaptiveASHAConfig{
			Metric:              "metric",
			SmallerIsBetter:     true,
			Divisor:             4,
			MaxRungs:            5,
			Mode:                StandardMode,
			MaxConcurrentTrials: 8,
		},
	}
	assert.DeepEqual(t, actual, expected)
}

func TestDefaultSmallerIsBetter(t *testing.T) {
	var actual1 = DefaultExperimentConfig(nil).Searcher
	assert.NilError(t, json.Unmarshal([]byte(`
{
  "name": "random",
  "metric": "metric",
  "smaller_is_better": true,
  "max_trials": 10,
  "max_length": {
    "batches": 10
  }
}
`), &actual1))

	expected1 := SearcherConfig{
		Metric:          "metric",
		SmallerIsBetter: true,
		RandomConfig: &RandomConfig{
			MaxTrials: 10,
			MaxLength: NewLengthInBatches(10),
		},
	}
	assert.DeepEqual(t, actual1, expected1)

	var actual2 = DefaultExperimentConfig(nil).Searcher
	assert.NilError(t, json.Unmarshal([]byte(`
{
  "name": "random",
  "metric": "metric",
  "max_trials": 10,
  "max_length": {
    "batches": 10
  }
}
`), &actual2))

	expected2 := SearcherConfig{
		Metric:          "metric",
		SmallerIsBetter: true,
		RandomConfig: &RandomConfig{
			MaxTrials: 10,
			MaxLength: NewLengthInBatches(10),
		},
	}
	assert.DeepEqual(t, actual2, expected2)

	var actual3 = DefaultExperimentConfig(nil).Searcher
	assert.NilError(t, json.Unmarshal([]byte(`
{
  "name": "random",
  "metric": "metric",
  "max_trials": 10,
  "smaller_is_better": false,
  "max_length": {
    "batches": 10
  }
}
`), &actual3))

	expected3 := SearcherConfig{
		Metric:          "metric",
		SmallerIsBetter: false,
		RandomConfig: &RandomConfig{
			MaxTrials: 10,
			MaxLength: NewLengthInBatches(10),
		},
	}
	assert.DeepEqual(t, actual3, expected3)
}

// TestAdaptiveASHABracketRungsConfig checks that adaptive_asha config bracket rungs are correct.
func TestAdaptiveASHABracketRungsConfig(t *testing.T) {
	json1 := []byte(`
{
  "name": "adaptive_asha",
  "metric": "metric",
  "max_length": {
	  "batches": 12800
  },
  "max_trials": 100,
  "bracket_rungs": [5, 10, 15, 20]
}
`)
	var actual SearcherConfig
	assert.NilError(t, json.Unmarshal(json1, &actual))

	expected := SearcherConfig{
		Metric: "metric",
		AdaptiveASHAConfig: &AdaptiveASHAConfig{
			Metric:       "metric",
			MaxLength:    NewLengthInBatches(12800),
			MaxTrials:    100,
			BracketRungs: []int{5, 10, 15, 20},
		},
	}
	assert.DeepEqual(t, actual, expected)
}

// TestPBTConfig tests basic serialization and deserialization of PBT config.
func TestPBTConfig(t *testing.T) {
	json1 := []byte(`
{
  "name": "pbt",
  "metric": "metric",
  "population_size": 101,
  "num_rounds": 102,
  "length_per_round": {
    "batches": 103
  },
  "replace_function": {
    "truncate_fraction": 0.17
  },
  "explore_function": {
    "resample_probability": 0.34,
    "perturb_factor": 0.51
  }
}
`)
	var actual SearcherConfig
	assert.NilError(t, json.Unmarshal(json1, &actual))

	expected := SearcherConfig{
		Metric: "metric",
		PBTConfig: &PBTConfig{
			Metric:         "metric",
			PopulationSize: 101,
			NumRounds:      102,
			LengthPerRound: NewLengthInBatches(103),
			PBTReplaceConfig: PBTReplaceConfig{
				TruncateFraction: .17,
			},
			PBTExploreConfig: PBTExploreConfig{
				ResampleProbability: .34,
				PerturbFactor:       .51,
			},
		},
	}
	assert.DeepEqual(t, actual, expected)
}

// TestLength tests basic serialization and deserialization of length.
func TestLength(t *testing.T) {
	testCases := []struct {
		name      string
		json      string
		expected  Length
		shouldErr bool
	}{
		{
			name:      "records",
			json:      "{\"records\":1000}",
			expected:  NewLength(Records, 1000),
			shouldErr: false,
		},
		{
			name:      "batches",
			json:      "{\"batches\":100}",
			expected:  NewLength(Batches, 100),
			shouldErr: false,
		},
		{
			name:      "epochs",
			json:      "{\"epochs\":1}",
			expected:  NewLength(Epochs, 1),
			shouldErr: false,
		},
		{
			name:      "invalid configuration -- more than one unit",
			json:      "{\"batches\":100,\"epochs\": 1}",
			shouldErr: true,
		},
		{
			name:      "invalid configuration -- no units",
			json:      "{}",
			shouldErr: true,
		},
	}

	for idx := range testCases {
		tc := testCases[idx]

		t.Run(tc.name, func(t *testing.T) {
			var actual Length
			err := json.Unmarshal([]byte(tc.json), &actual)
			if tc.shouldErr {
				assert.Error(t, err, fmt.Sprintf("invalid length: %s", tc.json))
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, actual, tc.expected)

			b, err := json.Marshal(actual)
			assert.NilError(t, err)
			assert.DeepEqual(t, tc.json, string(b))
		})
	}
}
