package searcher

import (
	"fmt"
	"math"
	"strings"

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

func unflattenSample(h hparamSample) hparamSample {
	result := make(hparamSample)
	for key, element := range h {
		nesting := strings.Split(key, ".")
		hPointer := result
		if len(nesting) > 1 {
			for i := 0; i < len(nesting)-1; i++ {
				k := nesting[i]
				if _, ok := hPointer[k]; !ok {
					hPointer[k] = make(map[string]interface{})
				}
				hPointer = hPointer[k].(map[string]interface{})
			}
		}
		hPointer[nesting[len(nesting)-1]] = element
	}
	return result
}

func sampleAll(h expconf.Hyperparameters, rand *nprand.State) hparamSample {
	flatSample := make(hparamSample)
	flatHPs := expconf.FlattenHPs(h)
	flatHPs.Each(func(name string, param expconf.Hyperparameter) {
		flatSample[name] = sampleOne(param, rand)
	})
	return unflattenSample(flatSample)
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
