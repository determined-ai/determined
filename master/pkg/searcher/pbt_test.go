package searcher

import (
	"math/rand"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestPBTSearcherWorkloads(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		// After the first round, trial 1 beats trial 2, spawning trial 3. Trial 1 lasts for two rounds
		// and the others last one round each.
		config := expconf.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: ptrs.BoolPtr(false),
			PopulationSize:  2,
			NumRounds:       2,
			LengthPerRound:  expconf.NewLengthInBatches(200),
			PBTReplaceConfig: expconf.PBTReplaceConfig{
				TruncateFraction: .5,
			},
			PBTExploreConfig: expconf.PBTExploreConfig{},
		}
		schemas.FillDefaults(&config)

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]Runnable{
			toOps("200B V C 200B V"),
			toOps("200B V"),
			toOps("200B V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})

	t.Run("no_truncation", func(t *testing.T) {
		// There is no truncation, so the initial population just survives forever.
		config := expconf.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: ptrs.BoolPtr(false),
			PopulationSize:  3,
			NumRounds:       4,
			LengthPerRound:  expconf.NewLengthInBatches(400),
			PBTReplaceConfig: expconf.PBTReplaceConfig{
				TruncateFraction: 0.,
			},
			PBTExploreConfig: expconf.PBTExploreConfig{},
		}
		schemas.FillDefaults(&config)

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]Runnable{
			toOps("400B V 400B V 400B V 400B V"),
			toOps("400B V 400B V 400B V 400B V"),
			toOps("400B V 400B V 400B V 400B V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})

	t.Run("even_odd", func(t *testing.T) {
		// After the first round, trial 1 beats trial 2, spawning trial 3. After the second round, trial 3
		// beats trial 1, spawning trial 4. Thus we have two trials that run for two rounds (1, 3) and two
		// that run for one round (2, 4).
		config := expconf.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: ptrs.BoolPtr(false),
			PopulationSize:  2,
			NumRounds:       3,
			LengthPerRound:  expconf.NewLengthInBatches(1700),
			PBTReplaceConfig: expconf.PBTReplaceConfig{
				TruncateFraction: .5,
			},
			PBTExploreConfig: expconf.PBTExploreConfig{},
		}
		schemas.FillDefaults(&config)

		val := func(random *rand.Rand, trialID, _ int) float64 {
			if trialID%2 == 0 {
				return -float64(trialID)
			}
			return float64(trialID)
		}

		expected := [][]Runnable{
			toOps("1700B V C 1700B V"),
			toOps("1700B V C 1700B V"),
			toOps("1700B V"),
			toOps("1700B V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})

	t.Run("new_is_better", func(t *testing.T) {
		// After each round, the two lowest-numbered trials are replaced by two new trials. Each trial
		// therefore lasts for two rounds, except for two of the initial population and the two created
		// right before the last round.
		config := expconf.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: ptrs.BoolPtr(false),
			PopulationSize:  4,
			NumRounds:       8,
			LengthPerRound:  expconf.NewLengthInBatches(500),
			PBTReplaceConfig: expconf.PBTReplaceConfig{
				TruncateFraction: .5,
			},
			PBTExploreConfig: expconf.PBTExploreConfig{},
		}
		schemas.FillDefaults(&config)

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]Runnable{
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V C 500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})

	t.Run("old_is_better", func(t *testing.T) {
		// Same as the above case, except that smaller is now better; thus, the two lowest-numbered trials
		// are always the best and survive forever, but all other trials last only one round.
		config := expconf.PBTConfig{
			Metric:          defaultMetric,
			SmallerIsBetter: ptrs.BoolPtr(true),
			PopulationSize:  4,
			NumRounds:       8,
			LengthPerRound:  expconf.NewLengthInBatches(500),
			PBTReplaceConfig: expconf.PBTReplaceConfig{
				TruncateFraction: .5,
			},
			PBTExploreConfig: expconf.PBTExploreConfig{},
		}
		schemas.FillDefaults(&config)

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]Runnable{
			toOps("500B V C 500B V C 500B V C 500B V C 500B V C 500B V C 500B V C 500B V"),
			toOps("500B V C 500B V C 500B V C 500B V C 500B V C 500B V C 500B V C 500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
			toOps("500B V"),
		}
		checkSimulation(t, newPBTSearch(config), nil, val, expected)
	})
}

func TestPBTSearcherReproducibility(t *testing.T) {
	conf := expconf.PBTConfig{
		Metric: defaultMetric, SmallerIsBetter: ptrs.BoolPtr(true),
		PopulationSize: 10, NumRounds: 10, LengthPerRound: expconf.NewLengthInBatches(1000),
		PBTReplaceConfig: expconf.PBTReplaceConfig{TruncateFraction: 0.5},
		PBTExploreConfig: expconf.PBTExploreConfig{ResampleProbability: 0.5, PerturbFactor: 0.5},
	}
	schemas.FillDefaults(&conf)
	searchMethod := func() SearchMethod { return newPBTSearch(conf) }
	checkReproducibility(t, searchMethod, nil, defaultMetric)
}

func testPBTExploreWithSeed(t *testing.T, seed uint32) {
	nullConfig := expconf.PBTConfig{
		Metric:           defaultMetric,
		SmallerIsBetter:  ptrs.BoolPtr(true),
		PopulationSize:   10,
		NumRounds:        10,
		LengthPerRound:   expconf.NewLengthInBatches(1000),
		PBTReplaceConfig: expconf.PBTReplaceConfig{},
		PBTExploreConfig: expconf.PBTExploreConfig{},
	}
	schemas.FillDefaults(&nullConfig)

	spec := expconf.Hyperparameters{
		"cat": expconf.Hyperparameter{
			CategoricalHyperparameter: &expconf.CategoricalHyperparameter{
				Vals: []interface{}{0, 1, 2, 3, 4, 5, 6},
			},
		},
		"const": expconf.Hyperparameter{
			ConstHyperparameter: &expconf.ConstHyperparameter{
				Val: "val",
			},
		},
		"double": expconf.Hyperparameter{
			DoubleHyperparameter: &expconf.DoubleHyperparameter{
				Minval: 0, Maxval: 100,
			},
		},
		"int": expconf.Hyperparameter{
			IntHyperparameter: &expconf.IntHyperparameter{
				Minval: 0, Maxval: 100,
			},
		},
		"log": expconf.Hyperparameter{
			LogHyperparameter: &expconf.LogHyperparameter{
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
		spec.Each(func(name string, _ expconf.Hyperparameter) {
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

// XXX: delete this after code stops changing so much.  It's good to have it for now.
func checkValidate(schema schemas.Schema) error {
	return schemas.IsComplete(schema)
}

func TestPBTValidation(t *testing.T) {
	goodConfig := expconf.SearcherConfig{
		PBTConfig: &expconf.PBTConfig{
			Metric:           defaultMetric,
			SmallerIsBetter:  ptrs.BoolPtr(true),
			PopulationSize:   10,
			NumRounds:        10,
			LengthPerRound:   expconf.NewLengthInBatches(1000),
			PBTReplaceConfig: expconf.PBTReplaceConfig{},
			PBTExploreConfig: expconf.PBTExploreConfig{},
		},
	}
	schemas.FillDefaults(&goodConfig)
	assert.NilError(t, checkValidate(&goodConfig))

	{
		badConfig := expconf.SearcherConfig{}
		schemas.Merge(&badConfig, goodConfig)
		badConfig.PBTConfig.PopulationSize = 0
		assert.ErrorContains(t, checkValidate(&badConfig), "population_size")
		badConfig.PBTConfig.PopulationSize = -1
		assert.ErrorContains(t, checkValidate(&badConfig), "population_size")
	}

	{
		badConfig := goodConfig
		badConfig.PBTConfig.NumRounds = 0
		assert.ErrorContains(t, checkValidate(&badConfig), "num_rounds")
		badConfig.PBTConfig.NumRounds = -1
		assert.ErrorContains(t, checkValidate(&badConfig), "num_rounds")
	}

	{
		badConfig := goodConfig
		badConfig.PBTConfig.LengthPerRound = expconf.NewLengthInBatches(0)
		assert.ErrorContains(t, checkValidate(&badConfig), "length_per_round")
		badConfig.PBTConfig.LengthPerRound = expconf.NewLengthInBatches(-1)
		assert.ErrorContains(t, checkValidate(&badConfig), "length_per_round")
	}

	{
		badConfig := goodConfig
		badConfig.PBTConfig.PerturbFactor = -.1
		assert.ErrorContains(t, checkValidate(&badConfig), "perturb_factor")
		badConfig.PBTConfig.PerturbFactor = 1.1
		assert.ErrorContains(t, checkValidate(&badConfig), "perturb_factor")
	}

	{
		badConfig := goodConfig
		badConfig.PBTConfig.TruncateFraction = -.1
		assert.ErrorContains(t, checkValidate(&badConfig), "truncate_fraction")
		badConfig.PBTConfig.TruncateFraction = .6
		assert.ErrorContains(t, checkValidate(&badConfig), "truncate_fraction")
	}

	{
		badConfig := goodConfig
		badConfig.PBTConfig.ResampleProbability = -.1
		assert.ErrorContains(t, checkValidate(&badConfig), "resample_probability")
		badConfig.PBTConfig.ResampleProbability = 1.1
		assert.ErrorContains(t, checkValidate(&badConfig), "resample_probability")
	}
}

func TestPBTSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newConstantPredefinedTrial(toOps("200B V C 200B V"), 0.5),
				newConstantPredefinedTrial(toOps("200B V"), 0.6),
				// Second generation beats first generation.
				newConstantPredefinedTrial(toOps("200B V C 200B V C 200B V"), 0.1),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(toOps("200B V"), 0.2),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(toOps("200B V"), 0.3),
			},
			config: expconf.SearcherConfig{
				PBTConfig: &expconf.PBTConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(true),
					PopulationSize:  2,
					NumRounds:       4,
					LengthPerRound:  expconf.NewLengthInBatches(200),
					PBTReplaceConfig: expconf.PBTReplaceConfig{
						TruncateFraction: .5,
					},
					PBTExploreConfig: expconf.PBTExploreConfig{},
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newEarlyExitPredefinedTrial(toOps("200B"), 0.5),
				newConstantPredefinedTrial(toOps("200B V C 200B V"), 0.6),
				// Second generation beats first generation.
				newConstantPredefinedTrial(toOps("200B V C 200B V C 200B V"), 0.1),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(toOps("200B V"), 0.2),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(toOps("200B V"), 0.3),
			},
			config: expconf.SearcherConfig{
				PBTConfig: &expconf.PBTConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(true),
					PopulationSize:  2,
					NumRounds:       4,
					LengthPerRound:  expconf.NewLengthInBatches(200),
					PBTReplaceConfig: expconf.PBTReplaceConfig{
						TruncateFraction: .5,
					},
					PBTExploreConfig: expconf.PBTExploreConfig{},
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newConstantPredefinedTrial(toOps("200B V C 200B V"), 0.5),
				newConstantPredefinedTrial(toOps("200B V"), 0.4),
				// Second generation beats first generation.
				newConstantPredefinedTrial(toOps("200B V C 200B V C 200B V"), 0.9),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(toOps("200B V"), 0.8),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(toOps("200B V"), 0.7),
			},
			config: expconf.SearcherConfig{
				PBTConfig: &expconf.PBTConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(false),
					PopulationSize:  2,
					NumRounds:       4,
					LengthPerRound:  expconf.NewLengthInBatches(200),
					PBTReplaceConfig: expconf.PBTReplaceConfig{
						TruncateFraction: .5,
					},
					PBTExploreConfig: expconf.PBTExploreConfig{},
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newEarlyExitPredefinedTrial(toOps("200B V C 200B"), 0.5),
				newConstantPredefinedTrial(toOps("200B V"), 0.4),
				// Second generation beats first generation.
				newConstantPredefinedTrial(toOps("200B V C 200B V C 200B V"), 0.9),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(toOps("200B V"), 0.8),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(toOps("200B V"), 0.7),
			},
			config: expconf.SearcherConfig{
				PBTConfig: &expconf.PBTConfig{
					Metric:          "error",
					SmallerIsBetter: ptrs.BoolPtr(false),
					PopulationSize:  2,
					NumRounds:       4,
					LengthPerRound:  expconf.NewLengthInBatches(200),
					PBTReplaceConfig: expconf.PBTReplaceConfig{
						TruncateFraction: .5,
					},
					PBTExploreConfig: expconf.PBTExploreConfig{},
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
