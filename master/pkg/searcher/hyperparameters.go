package searcher

import (
	"fmt"
	"math"

	"github.com/determined-ai/determined/master/pkg/nprand"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// HParamSample is a sampling of the hyperparameters for a model.
type HParamSample map[string]interface{}

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
		return p.Minval() + rand.Intn(p.Maxval()-p.Minval()+1)
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
