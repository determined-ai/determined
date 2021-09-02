package searcher

import (
	"math/rand"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func replaceConfigPtr(val expconf.PBTReplaceConfig) *expconf.PBTReplaceConfig {
	return &val
}

func exploreConfigPtr(val expconf.PBTExploreConfig) *expconf.PBTExploreConfig {
	return &val
}

func TestPBTSearcherWorkloads(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		// After the first round, trial 1 beats trial 2, spawning trial 3. Trial 1 lasts for two rounds
		// and the others last one round each.
		config := expconf.PBTConfig{
			RawPopulationSize: ptrs.IntPtr(2),
			RawNumRounds:      ptrs.IntPtr(2),
			RawLengthPerRound: lengthPtr(expconf.NewLengthInBatches(200)),
			RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{
				TruncateFraction: .5,
			}),
			RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]ValidateAfter{
			toOps("200B 400B"),
			toOps("200B"),
			toOps("200B"),
		}
		checkSimulation(t, newPBTSearch(config, false), nil, val, expected)
	})

	t.Run("no_truncation", func(t *testing.T) {
		// There is no truncation, so the initial population just survives forever.
		config := expconf.PBTConfig{
			RawPopulationSize: ptrs.IntPtr(3),
			RawNumRounds:      ptrs.IntPtr(4),
			RawLengthPerRound: lengthPtr(expconf.NewLengthInBatches(400)),
			RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{
				TruncateFraction: 0.,
			}),
			RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]ValidateAfter{
			toOps("400B 800B 1200B 1600B"),
			toOps("400B 800B 1200B 1600B"),
			toOps("400B 800B 1200B 1600B"),
		}
		checkSimulation(t, newPBTSearch(config, false), nil, val, expected)
	})

	t.Run("even_odd", func(t *testing.T) {
		// After the first round, trial 1 beats trial 2, spawning trial 3. After the second round, trial 3
		// beats trial 1, spawning trial 4. Thus we have two trials that run for two rounds (1, 3) and two
		// that run for one round (2, 4).
		config := expconf.PBTConfig{
			RawPopulationSize: ptrs.IntPtr(2),
			RawNumRounds:      ptrs.IntPtr(3),
			RawLengthPerRound: lengthPtr(expconf.NewLengthInBatches(1700)),
			RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{
				TruncateFraction: .5,
			}),
			RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			if trialID%2 == 0 {
				return -float64(trialID)
			}
			return float64(trialID)
		}

		expected := [][]ValidateAfter{
			toOps("1700B 3400B"),
			toOps("1700B 3400B"),
			toOps("1700B"),
			toOps("1700B"),
		}
		checkSimulation(t, newPBTSearch(config, false), nil, val, expected)
	})

	t.Run("new_is_better", func(t *testing.T) {
		// After each round, the two lowest-numbered trials are replaced by two new trials. Each trial
		// therefore lasts for two rounds, except for two of the initial population and the two created
		// right before the last round.
		config := expconf.PBTConfig{
			RawPopulationSize: ptrs.IntPtr(4),
			RawNumRounds:      ptrs.IntPtr(8),
			RawLengthPerRound: lengthPtr(expconf.NewLengthInBatches(500)),
			RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{
				TruncateFraction: .5,
			}),
			RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]ValidateAfter{
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B 1000B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
		}
		checkSimulation(t, newPBTSearch(config, false), nil, val, expected)
	})

	t.Run("old_is_better", func(t *testing.T) {
		// Same as the above case, except that smaller is now better; thus, the two lowest-numbered trials
		// are always the best and survive forever, but all other trials last only one round.
		config := expconf.PBTConfig{
			RawPopulationSize: ptrs.IntPtr(4),
			RawNumRounds:      ptrs.IntPtr(8),
			RawLengthPerRound: lengthPtr(expconf.NewLengthInBatches(500)),
			RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{
				TruncateFraction: .5,
			}),
			RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
		}

		val := func(random *rand.Rand, trialID, _ int) float64 {
			return float64(trialID)
		}

		expected := [][]ValidateAfter{
			toOps("500B 1000B 1500B 2000B 2500B 3000B 3500B 4000B"),
			toOps("500B 1000B 1500B 2000B 2500B 3000B 3500B 4000B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
			toOps("500B"),
		}
		checkSimulation(t, newPBTSearch(config, true), nil, val, expected)
	})
}

func TestPBTSearcherReproducibility(t *testing.T) {
	conf := expconf.PBTConfig{
		RawPopulationSize:  ptrs.IntPtr(10),
		RawNumRounds:       ptrs.IntPtr(10),
		RawLengthPerRound:  lengthPtr(expconf.NewLengthInBatches(1000)),
		RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{TruncateFraction: 0.5}),
		RawExploreFunction: exploreConfigPtr(
			expconf.PBTExploreConfig{ResampleProbability: 0.5, PerturbFactor: 0.5},
		),
	}
	searchMethod := func() SearchMethod { return newPBTSearch(conf, true) }
	checkReproducibility(t, searchMethod, nil, defaultMetric)
}

func testPBTExploreWithSeed(t *testing.T, seed uint32) {
	nullConfig := func() expconf.PBTConfig {
		return expconf.PBTConfig{
			RawPopulationSize:  ptrs.IntPtr(10),
			RawNumRounds:       ptrs.IntPtr(10),
			RawLengthPerRound:  lengthPtr(expconf.NewLengthInBatches(1000)),
			RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{}),
			RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
		}
	}

	spec := expconf.Hyperparameters{
		"cat": expconf.Hyperparameter{
			RawCategoricalHyperparameter: &expconf.CategoricalHyperparameter{
				RawVals: []interface{}{0, 1, 2, 3, 4, 5, 6},
			},
		},
		"nested": expconf.Hyperparameter{
			RawNestedHyperparameter: &map[string]expconf.Hyperparameter{
				"hp1": {
					RawCategoricalHyperparameter: &expconf.CategoricalHyperparameter{
						RawVals: []interface{}{0, 1, 2, 3, 4, 5, 6},
					},
				},
			},
		},
		"const": expconf.Hyperparameter{
			RawConstHyperparameter: &expconf.ConstHyperparameter{
				RawVal: "val",
			},
		},
		"double": expconf.Hyperparameter{
			RawDoubleHyperparameter: &expconf.DoubleHyperparameter{
				RawMinval: 0, RawMaxval: 100,
			},
		},
		"int": expconf.Hyperparameter{
			RawIntHyperparameter: &expconf.IntHyperparameter{
				RawMinval: 0, RawMaxval: 100,
			},
		},
		"log": expconf.Hyperparameter{
			RawLogHyperparameter: &expconf.LogHyperparameter{
				RawBase: 10, RawMinval: -4, RawMaxval: -2,
			},
		},
	}
	sample := HParamSample{
		"cat": 3,
		"nested": map[string]int{
			"hp1": 0,
		},
		"const":  "val",
		"double": 50.,
		"int":    50,
		"log":    .001,
	}

	ctx := context{rand: nprand.New(seed), hparams: spec}

	// Test that exploring with no resampling and no perturbing does not change the hyperparameters.
	{
		pbt := newPBTSearch(nullConfig(), true).(*pbtSearch)
		newSample := pbt.exploreParams(ctx, sample)
		assert.DeepEqual(t, sample, newSample)
	}

	// Test that exploring with guaranteed resampling changes all of the hyperparameters.
	{
		resamplingConfig := nullConfig()
		resamplingConfig.RawExploreFunction.ResampleProbability = 1

		// Create a hyperparameter sample where none of the values are actually valid, then resample it.
		invalidSample := make(HParamSample)
		spec.Each(func(name string, _ expconf.Hyperparameter) {
			invalidSample[name] = nil
		})
		pbt := newPBTSearch(nullConfig(), true).(*pbtSearch)
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
		perturbingConfig := nullConfig()
		perturbingConfig.RawExploreFunction.PerturbFactor = .5
		pbt := newPBTSearch(perturbingConfig, true).(*pbtSearch)

		newSample := pbt.exploreParams(ctx, sample)

		assert.Equal(t, len(sample), len(newSample))

		// Check that only the numerical hyperparameters have changed.
		assert.Equal(t, sample["cat"], newSample["cat"])
		assert.Equal(t, sample["const"], newSample["const"])
		assert.DeepEqual(t, sample["nested"], newSample["nested"])
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
	for i := uint32(0); i < 1; i++ {
		testPBTExploreWithSeed(t, i)
	}
}

func TestPBTSearchMethod(t *testing.T) {
	testCases := []valueSimulationTestCase{
		{
			name: "smaller is better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newConstantPredefinedTrial(toOps("200B 400B"), 0.5),
				newConstantPredefinedTrial(toOps("200B"), 0.6),
				// Second generation beats first generation.
				newConstantPredefinedTrial(toOps("200B 400B 600B"), 0.1),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(toOps("200B"), 0.2),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(toOps("200B"), 0.3),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(true),
				RawPBTConfig: &expconf.PBTConfig{
					RawPopulationSize: ptrs.IntPtr(2),
					RawNumRounds:      ptrs.IntPtr(4),
					RawLengthPerRound: lengthPtr(expconf.NewLengthInBatches(200)),
					RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{
						TruncateFraction: .5,
					}),
					RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
				},
			},
		},
		{
			name: "early exit -- smaller is better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newEarlyExitPredefinedTrial(toOps("200B"), 0.5),
				newConstantPredefinedTrial(toOps("200B 400B"), 0.6),
				// Second generation beats first generation.
				newConstantPredefinedTrial(toOps("200B 400B 600B"), 0.1),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(toOps("200B"), 0.2),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(toOps("200B"), 0.3),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(true),
				RawPBTConfig: &expconf.PBTConfig{
					RawPopulationSize: ptrs.IntPtr(2),
					RawNumRounds:      ptrs.IntPtr(4),
					RawLengthPerRound: lengthPtr(expconf.NewLengthInBatches(200)),
					RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{
						TruncateFraction: .5,
					}),
					RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
				},
			},
		},
		{
			name: "smaller is not better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newConstantPredefinedTrial(toOps("200B 400B"), 0.5),
				newConstantPredefinedTrial(toOps("200B"), 0.4),
				// Second generation beats first generation.
				newConstantPredefinedTrial(toOps("200B 400B 600B"), 0.9),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(toOps("200B"), 0.8),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(toOps("200B"), 0.7),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(false),
				RawPBTConfig: &expconf.PBTConfig{
					RawPopulationSize: ptrs.IntPtr(2),
					RawNumRounds:      ptrs.IntPtr(4),
					RawLengthPerRound: lengthPtr(expconf.NewLengthInBatches(200)),
					RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{
						TruncateFraction: .5,
					}),
					RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
				},
			},
		},
		{
			name: "early exit -- smaller is not better",
			expectedTrials: []predefinedTrial{
				// First generation.
				newEarlyExitPredefinedTrial(toOps("200B 400B"), 0.5),
				newConstantPredefinedTrial(toOps("200B"), 0.4),
				// Second generation beats first generation.
				newConstantPredefinedTrial(toOps("200B 400B 600B"), 0.9),
				// Third generation loses to second generation.
				newConstantPredefinedTrial(toOps("200B"), 0.8),
				// Fourth generation loses to second generation also.
				newConstantPredefinedTrial(toOps("200B"), 0.7),
			},
			config: expconf.SearcherConfig{
				RawSmallerIsBetter: ptrs.BoolPtr(false),
				RawPBTConfig: &expconf.PBTConfig{
					RawPopulationSize: ptrs.IntPtr(2),
					RawNumRounds:      ptrs.IntPtr(4),
					RawLengthPerRound: lengthPtr(expconf.NewLengthInBatches(200)),
					RawReplaceFunction: replaceConfigPtr(expconf.PBTReplaceConfig{
						TruncateFraction: .5,
					}),
					RawExploreFunction: exploreConfigPtr(expconf.PBTExploreConfig{}),
				},
			},
		},
	}

	runValueSimulationTestCases(t, testCases)
}
