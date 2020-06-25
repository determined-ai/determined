package model

import (
	"encoding/json"
	"testing"

	"gotest.tools/assert"
)

func TestASHAMaxConcurrentTrials(t *testing.T) {
	var actual = DefaultExperimentConfig().Searcher
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
	var actual1 = DefaultExperimentConfig().Searcher
	assert.NilError(t, json.Unmarshal([]byte(`
{
  "name": "adaptive_simple",
  "metric": "metric",
  "smaller_is_better": true
}
`), &actual1))

	expected1 := SearcherConfig{
		Metric:          "metric",
		SmallerIsBetter: true,
		AdaptiveSimpleConfig: &AdaptiveSimpleConfig{
			Metric:          "metric",
			SmallerIsBetter: true,
			Divisor:         4,
			MaxRungs:        5,
			Mode:            StandardMode,
		},
	}
	assert.DeepEqual(t, actual1, expected1)

	var actual2 = DefaultExperimentConfig().Searcher
	assert.NilError(t, json.Unmarshal([]byte(`
{
  "name": "adaptive_simple",
  "metric": "metric"
}
`), &actual2))

	expected2 := SearcherConfig{
		Metric:          "metric",
		SmallerIsBetter: true,
		AdaptiveSimpleConfig: &AdaptiveSimpleConfig{
			Metric:          "metric",
			SmallerIsBetter: true,
			Divisor:         4,
			MaxRungs:        5,
			Mode:            StandardMode,
		},
	}
	assert.DeepEqual(t, actual2, expected2)

	var actual3 = DefaultExperimentConfig().Searcher
	assert.NilError(t, json.Unmarshal([]byte(`
{
  "name": "adaptive_simple",
  "metric": "metric",
  "smaller_is_better": false
}
`), &actual3))

	expected3 := SearcherConfig{
		Metric:          "metric",
		SmallerIsBetter: false,
		AdaptiveSimpleConfig: &AdaptiveSimpleConfig{
			Metric:          "metric",
			SmallerIsBetter: false,
			Divisor:         4,
			MaxRungs:        5,
			Mode:            StandardMode,
		},
	}
	assert.DeepEqual(t, actual3, expected3)
}

// TestAdaptiveBracketRungsConfig checks that adaptive config bracket rungs are loaded correctly.
func TestAdaptiveBracketRungsConfig(t *testing.T) {
	json1 := []byte(`
{
  "name": "adaptive",
  "metric": "metric",
  "target_trial_steps": 128,
  "step_budget": 100,
  "bracket_rungs": [5, 10, 15, 20]
}
`)
	var actual SearcherConfig
	assert.NilError(t, json.Unmarshal(json1, &actual))

	expected := SearcherConfig{
		Metric: "metric",
		AdaptiveConfig: &AdaptiveConfig{
			Metric:           "metric",
			TargetTrialSteps: 128,
			StepBudget:       100,
			BracketRungs:     []int{5, 10, 15, 20},
		},
	}
	assert.DeepEqual(t, actual, expected)
}

// TestAdaptiveASHABracketRungsConfig checks that adaptive_asha config bracket rungs are correct.
func TestAdaptiveASHABracketRungsConfig(t *testing.T) {
	json1 := []byte(`
{
  "name": "adaptive_asha",
  "metric": "metric",
  "target_trial_steps": 128,
  "max_trials": 100,
  "bracket_rungs": [5, 10, 15, 20]
}
`)
	var actual SearcherConfig
	assert.NilError(t, json.Unmarshal(json1, &actual))

	expected := SearcherConfig{
		Metric: "metric",
		AdaptiveASHAConfig: &AdaptiveASHAConfig{
			Metric:           "metric",
			TargetTrialSteps: 128,
			MaxTrials:        100,
			BracketRungs:     []int{5, 10, 15, 20},
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
  "steps_per_round": 103,
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
			StepsPerRound:  103,
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
