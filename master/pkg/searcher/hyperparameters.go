package searcher

import (
	"fmt"
	"math"

	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

type hparamSample map[string]interface{}

func (h hparamSample) GlobalBatchSize() int {
	// If the hyperparameters.global_batch_size is configured as a const hyperparameter,
	// we infer its type to be a float but in some cases, its type can be specified and an
	// int is also valid.
	f, ok := h[expconf.GlobalBatchSize].(float64)
	if ok {
		return int(f)
	}
	return h[expconf.GlobalBatchSize].(int)
}

func sampleAll(h expconf.Hyperparameters, rand *nprand.State) hparamSample {
	results := make(hparamSample)
	h.Each(func(name string, param expconf.Hyperparameter) {
		results[name] = sampleOne(param, rand)
	})
	return results
}

func sampleOne(h expconf.Hyperparameter, rand *nprand.State) interface{} {
	switch {
	case h.ConstHyperparameter != nil:
		p := h.ConstHyperparameter
		return p.Val
	case h.IntHyperparameter != nil:
		p := h.IntHyperparameter
		return p.Minval + rand.Intn(p.Maxval-p.Minval)
	case h.DoubleHyperparameter != nil:
		p := h.DoubleHyperparameter
		return rand.Uniform(p.Minval, p.Maxval)
	case h.LogHyperparameter != nil:
		p := h.LogHyperparameter
		return math.Pow(p.Base, rand.Uniform(p.Minval, p.Maxval))
	case h.CategoricalHyperparameter != nil:
		p := h.CategoricalHyperparameter
		return p.Vals[rand.Intn(len(p.Vals))]
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
