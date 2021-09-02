package searcher

import (
	"fmt"
	"math"

	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// HParamSample is a sampling of the hyperparameters for a model.
type HParamSample map[string]interface{}

// GlobalBatchSize returns the `global_batch_size` hyperparameter as an integer.
func (h HParamSample) GlobalBatchSize() int {
	// If the hyperparameters.global_batch_size is configured as a const hyperparameter,
	// we infer its type to be a float but in some cases, its type can be specified and an
	// int is also valid.
	f, ok := h[expconf.GlobalBatchSize].(float64)
	if ok {
		return int(f)
	}
	return h[expconf.GlobalBatchSize].(int)
}

func sampleAll(h expconf.Hyperparameters, rand *nprand.State) HParamSample {
	results := make(HParamSample)
	h.Each(func(name string, param expconf.Hyperparameter) {
		results[name] = sampleOne(param, rand)
	})
	return results
}

func sampleOne(h expconf.Hyperparameter, rand *nprand.State) interface{} {
	switch {
	case h.RawConstHyperparameter != nil:
		p := h.RawConstHyperparameter
		return p.Val()
	case h.RawIntHyperparameter != nil:
		p := h.RawIntHyperparameter
		return p.Minval() + rand.Intn(p.Maxval()-p.Minval())
	case h.RawDoubleHyperparameter != nil:
		p := h.RawDoubleHyperparameter
		return rand.Uniform(p.Minval(), p.Maxval())
	case h.RawLogHyperparameter != nil:
		p := h.RawLogHyperparameter
		return math.Pow(p.Base(), rand.Uniform(p.Minval(), p.Maxval()))
	case h.RawCategoricalHyperparameter != nil:
		p := h.RawCategoricalHyperparameter
		return p.Vals()[rand.Intn(len(p.Vals()))]
	case h.RawNestedHyperparameter != nil:
		p := make(map[string]interface{})
		for key, val := range *h.RawNestedHyperparameter {
			p[key] = sampleOne(val, rand)
		}
		return p
	default:
		panic(fmt.Sprintf("unexpected hyperparameter type: %+v", h))
	}
}

func intClamp(val, minval, maxval int) int {
	switch {
	case val < minval:
		return minval
	case val > maxval:
		return maxval
	default:
		return val
	}
}

func doubleClamp(val, minval, maxval float64) float64 {
	switch {
	case val < minval:
		return minval
	case val > maxval:
		return maxval
	default:
		return val
	}
}
