package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestSamplingReproducibility(t *testing.T) {
	spec := expconf.Hyperparameters{
		"cat": {CategoricalHyperparameter: &expconf.CategoricalHyperparameter{
			Vals: []interface{}{0, 1, 2, 3, 4, 5, 6}}},
		"const":  {ConstHyperparameter: &expconf.ConstHyperparameter{Val: "val"}},
		"double": {DoubleHyperparameter: &expconf.DoubleHyperparameter{Minval: 0, Maxval: 100}},
		"int":    {IntHyperparameter: &expconf.IntHyperparameter{Minval: 0, Maxval: 100}},
		"log":    {LogHyperparameter: &expconf.LogHyperparameter{Base: 10, Minval: -2, Maxval: 2}},
	}

	// Run the checks multiple times; map iteration order, if it has an effect on the result, is an
	// uncontrollable source of randomness that may end up allowing the checks to pass incorrectly.
	for seed := uint32(0); seed < 50; seed++ {
		// Check that sampling twice gives the same result both times.
		rand1 := nprand.New(seed)
		rand2 := nprand.New(seed)

		sample1 := sampleAll(spec, rand1)
		sample2 := sampleAll(spec, rand2)

		assert.Equal(t, 5, len(sample1))
		assert.Equal(t, 5, len(sample2))

		for name, val1 := range sample1 {
			val2, ok := sample2[name]
			assert.Assert(t, ok)
			assert.Equal(t, val1, val2)
		}

		// Compare direct RNG output to check by proxy whether the internal states have stayed the same.
		assert.Equal(t, rand1.Bits64(), rand2.Bits64())
	}
}
