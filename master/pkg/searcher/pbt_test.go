package searcher

import (
	"math/rand"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

func TestPBTSearcherWorkloads(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		// After the first round, trial 1 beats trial 2, spawning trial 3. Trial 1 lasts for two rounds
		// and the others last one round each.
		config := model.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: false,
			PopulationSize:  2,
			NumRounds:       2,
			StepsPerRound:   2,
			PBTReplaceConfig: model.PBTReplaceConfig{
				TruncateFraction: .5,
			},
			PBTExploreConfig: model.PBTExploreConfig{},
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]Kind{
			toKinds("2S 1V 1C 2S 1V"),
			toKinds("2S 1V"),
			toKinds("2S 1V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})

	t.Run("no_truncation", func(t *testing.T) {
		// There is no truncation, so the initial population just survives forever.
		config := model.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: false,
			PopulationSize:  3,
			NumRounds:       4,
			StepsPerRound:   4,
			PBTReplaceConfig: model.PBTReplaceConfig{
				TruncateFraction: 0.,
			},
			PBTExploreConfig: model.PBTExploreConfig{},
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]Kind{
			toKinds("4S 1V 4S 1V 4S 1V 4S 1V"),
			toKinds("4S 1V 4S 1V 4S 1V 4S 1V"),
			toKinds("4S 1V 4S 1V 4S 1V 4S 1V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})

	t.Run("even_odd", func(t *testing.T) {
		// After the first round, trial 1 beats trial 2, spawning trial 3. After the second round, trial 3
		// beats trial 1, spawning trial 4. Thus we have two trials that run for two rounds (1, 3) and two
		// that run for one round (2, 4).
		config := model.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: false,
			PopulationSize:  2,
			NumRounds:       3,
			StepsPerRound:   17,
			PBTReplaceConfig: model.PBTReplaceConfig{
				TruncateFraction: .5,
			},
			PBTExploreConfig: model.PBTExploreConfig{},
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			if trialID%2 == 0 {
				return -float64(trialID)
			}
			return float64(trialID)
		}

		expected := [][]Kind{
			toKinds("17S 1V 1C 17S 1V"),
			toKinds("17S 1V 1C 17S 1V"),
			toKinds("17S 1V"),
			toKinds("17S 1V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})

	t.Run("new_is_better", func(t *testing.T) {
		// After each round, the two lowest-numbered trials are replaced by two new trials. Each trial
		// therefore lasts for two rounds, except for two of the initial population and the two created
		// right before the last round.
		config := model.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: false,
			PopulationSize:  4,
			NumRounds:       8,
			StepsPerRound:   5,
			PBTReplaceConfig: model.PBTReplaceConfig{
				TruncateFraction: .5,
			},
			PBTExploreConfig: model.PBTExploreConfig{},
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]Kind{
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})

	t.Run("old_is_better", func(t *testing.T) {
		// Same as the above case, except that smaller is now better; thus, the two lowest-numbered trials
		// are always the best and survive forever, but all other trials last only one round.
		config := model.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: true,
			PopulationSize:  4,
			NumRounds:       8,
			StepsPerRound:   5,
			PBTReplaceConfig: model.PBTReplaceConfig{
				TruncateFraction: .5,
			},
			PBTExploreConfig: model.PBTExploreConfig{},
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]Kind{
			toKinds("5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V"),
			toKinds("5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V 1C 5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
			toKinds("5S 1V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})
}

func TestPBTSearcherReproducibility(t *testing.T) {
	conf := model.PBTConfig{
		Metric: defaultMetric, SmallerIsBetter: true,
		PopulationSize: 10, NumRounds: 10, StepsPerRound: 10,
		PBTReplaceConfig: model.PBTReplaceConfig{TruncateFraction: 0.5},
		PBTExploreConfig: model.PBTExploreConfig{ResampleProbability: 0.5, PerturbFactor: 0.5},
	}
	checkReproducibility(t, func() SearchMethod { return newPBTSearch(conf) }, nil, defaultMetric)
}

func testPBTExploreWithSeed(t *testing.T, seed uint32) {
	nullConfig := model.PBTConfig{
		Metric:           defaultMetric,
		SmallerIsBetter:  true,
		PopulationSize:   10,
		NumRounds:        10,
		StepsPerRound:    10,
		PBTReplaceConfig: model.PBTReplaceConfig{},
		PBTExploreConfig: model.PBTExploreConfig{},
	}

	spec := model.Hyperparameters{
		"cat": model.Hyperparameter{
			CategoricalHyperparameter: &model.CategoricalHyperparameter{
				Vals: []interface{}{0, 1, 2, 3, 4, 5, 6},
			},
		},
		"const": model.Hyperparameter{
			ConstHyperparameter: &model.ConstHyperparameter{
				Val: "val",
			},
		},
		"double": model.Hyperparameter{
			DoubleHyperparameter: &model.DoubleHyperparameter{
				Minval: 0, Maxval: 100,
			},
		},
		"int": model.Hyperparameter{
			IntHyperparameter: &model.IntHyperparameter{
				Minval: 0, Maxval: 100,
			},
		},
		"log": model.Hyperparameter{
			LogHyperparameter: &model.LogHyperparameter{
				Base: 10, Minval: -4, Maxval: -2,
			},
		},
	}
	sample := hparamSample{
		"cat":    3,
		"const":  "val",
		"double": 50.,
		"int":    50,
		"log":    .001,
	}

	ctx := context{rand: nprand.New(seed), hparams: spec}

	// Test that exploring with no resampling and no perturbing does not change the hyperparameters.
	{
		pbt := newPBTSearch(nullConfig).(*pbtSearch)
		newSample := pbt.exploreParams(ctx, sample)
		assert.DeepEqual(t, sample, newSample)
	}

	// Test that exploring with guaranteed resampling changes all of the hyperparameters.
	{
		resamplingConfig := nullConfig
		resamplingConfig.ResampleProbability = 1

		// Create a hyperparameter sample where none of the values are actually valid, then resample it.
		invalidSample := make(hparamSample)
		spec.Each(func(name string, _ model.Hyperparameter) {
			invalidSample[name] = nil
		})
		pbt := newPBTSearch(nullConfig).(*pbtSearch)
		newSample := pbt.exploreParams(ctx, sample)

		assert.Equal(t, len(invalidSample), len(newSample))
		for name := range invalidSample {
			val, ok := newSample[name]
			assert.Assert(t, ok)
			assert.Assert(t, val != nil)
		}
	}

	// Test that guaranteed perturbing produces reasonable values.
	{
		perturbingConfig := nullConfig
		perturbingConfig.PerturbFactor = .5
		pbt := newPBTSearch(perturbingConfig).(*pbtSearch)

		newSample := pbt.exploreParams(ctx, sample)

		assert.Equal(t, len(sample), len(newSample))

		// Check that only the numerical hyperparameters have changed.
		assert.Equal(t, sample["cat"], newSample["cat"])
		assert.Equal(t, sample["const"], newSample["const"])
		assert.Assert(t, sample["double"] != newSample["double"])
		assert.Assert(t, sample["int"] != newSample["int"])
		assert.Assert(t, sample["log"] != newSample["log"])

		// Check that the new numerical values are in a reasonable range.
		assert.Assert(t, newSample["log"].(float64) >= 1e-4)
		assert.Assert(t, newSample["log"].(float64) <= 1e-2)
		assert.Assert(t, newSample["int"].(int) >= 0)
		assert.Assert(t, newSample["int"].(int) <= 100)
		assert.Assert(t, newSample["double"].(float64) >= 0)
		assert.Assert(t, newSample["double"].(float64) <= 100)
	}
}

func TestPBTExplore(t *testing.T) {
	for i := uint32(0); i < 100; i++ {
		testPBTExploreWithSeed(t, i)
	}
}

func TestPBTValidation(t *testing.T) {
	goodConfig := model.PBTConfig{
		Metric:           defaultMetric,
		SmallerIsBetter:  true,
		PopulationSize:   10,
		NumRounds:        10,
		StepsPerRound:    10,
		PBTReplaceConfig: model.PBTReplaceConfig{},
		PBTExploreConfig: model.PBTExploreConfig{},
	}
	assert.NilError(t, check.Validate(goodConfig))

	{
		badConfig := goodConfig
		badConfig.PopulationSize = 0
		assert.ErrorContains(t, check.Validate(badConfig), "population_size")
		badConfig.PopulationSize = -1
		assert.ErrorContains(t, check.Validate(badConfig), "population_size")
	}

	{
		badConfig := goodConfig
		badConfig.NumRounds = 0
		assert.ErrorContains(t, check.Validate(badConfig), "num_rounds")
		badConfig.NumRounds = -1
		assert.ErrorContains(t, check.Validate(badConfig), "num_rounds")
	}

	{
		badConfig := goodConfig
		badConfig.StepsPerRound = 0
		assert.ErrorContains(t, check.Validate(badConfig), "steps_per_round")
		badConfig.StepsPerRound = -1
		assert.ErrorContains(t, check.Validate(badConfig), "steps_per_round")
	}

	{
		badConfig := goodConfig
		badConfig.PerturbFactor = -.1
		assert.ErrorContains(t, check.Validate(badConfig), "perturb_factor")
		badConfig.PerturbFactor = 1.1
		assert.ErrorContains(t, check.Validate(badConfig), "perturb_factor")
	}

	{
		badConfig := goodConfig
		badConfig.TruncateFraction = -.1
		assert.ErrorContains(t, check.Validate(badConfig), "truncate_fraction")
		badConfig.TruncateFraction = .6
		assert.ErrorContains(t, check.Validate(badConfig), "truncate_fraction")
	}

	{
		badConfig := goodConfig
		badConfig.ResampleProbability = -.1
		assert.ErrorContains(t, check.Validate(badConfig), "resample_probability")
		badConfig.ResampleProbability = 1.1
		assert.ErrorContains(t, check.Validate(badConfig), "resample_probability")
	}
}

func TestPBTSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newConstantPredefinedTrial(0.5, 4, []int{2, 4}, []int{2}),
				newConstantPredefinedTrial(0.6, 2, []int{2}, nil),
				// Second generation beats first generation.
				newConstantPredefinedTrial(0.1, 6, []int{2, 4, 6}, []int{2, 4}),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(0.2, 2, []int{2}, nil),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(0.3, 2, []int{2}, nil),
			},
			config: model.SearcherConfig{
				PBTConfig: &model.PBTConfig{
					Metric:          "error",
					SmallerIsBetter: true,
					PopulationSize:  2,
					NumRounds:       4,
					StepsPerRound:   2,
					PBTReplaceConfig: model.PBTReplaceConfig{
						TruncateFraction: .5,
					},
					PBTExploreConfig: model.PBTExploreConfig{},
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newEarlyExitPredefinedTrial(0.5, 4, []int{2}, []int{2}),
				newConstantPredefinedTrial(0.6, 2, []int{2}, nil),
				// Second generation beats first generation.
				newConstantPredefinedTrial(0.1, 6, []int{2, 4, 6}, []int{2, 4}),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(0.2, 2, []int{2}, nil),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(0.3, 2, []int{2}, nil),
			},
			config: model.SearcherConfig{
				PBTConfig: &model.PBTConfig{
					Metric:          "error",
					SmallerIsBetter: true,
					PopulationSize:  2,
					NumRounds:       4,
					StepsPerRound:   2,
					PBTReplaceConfig: model.PBTReplaceConfig{
						TruncateFraction: .5,
					},
					PBTExploreConfig: model.PBTExploreConfig{},
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newConstantPredefinedTrial(0.5, 4, []int{2, 4}, []int{2}),
				newConstantPredefinedTrial(0.4, 2, []int{2}, nil),
				// Second generation beats first generation.
				newConstantPredefinedTrial(0.9, 6, []int{2, 4, 6}, []int{2, 4}),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(0.8, 2, []int{2}, nil),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(0.7, 2, []int{2}, nil),
			},
			config: model.SearcherConfig{
				PBTConfig: &model.PBTConfig{
					Metric:          "error",
					SmallerIsBetter: false,
					PopulationSize:  2,
					NumRounds:       4,
					StepsPerRound:   2,
					PBTReplaceConfig: model.PBTReplaceConfig{
						TruncateFraction: .5,
					},
					PBTExploreConfig: model.PBTExploreConfig{},
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newEarlyExitPredefinedTrial(0.5, 4, []int{2}, []int{2}),
				newConstantPredefinedTrial(0.4, 2, []int{2}, nil),
				// Second generation beats first generation.
				newConstantPredefinedTrial(0.9, 6, []int{2, 4, 6}, []int{2, 4}),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(0.8, 2, []int{2}, nil),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(0.7, 2, []int{2}, nil),
			},
			config: model.SearcherConfig{
				PBTConfig: &model.PBTConfig{
					Metric:          "error",
					SmallerIsBetter: false,
					PopulationSize:  2,
					NumRounds:       4,
					StepsPerRound:   2,
					PBTReplaceConfig: model.PBTReplaceConfig{
						TruncateFraction: .5,
					},
					PBTExploreConfig: model.PBTExploreConfig{},
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
