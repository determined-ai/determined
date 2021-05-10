package expconf

import (
	"encoding/json"
	"sort"

	"github.com/determined-ai/determined/master/pkg/union"
)

// GlobalBatchSize is the name of the hyperparameter for global_batch_size.
const GlobalBatchSize = "global_batch_size"

//go:generate ../gen.sh
// HyperparametersV0 is a versioned hyperparameters config.
type HyperparametersV0 map[string]HyperparameterV0

//go:generate ../gen.sh
// HyperparameterV0 is a sum type for hyperparameters.
type HyperparameterV0 struct {
	RawConstHyperparameter       *ConstHyperparameterV0       `union:"type,const" json:"-"`
	RawIntHyperparameter         *IntHyperparameterV0         `union:"type,int" json:"-"`
	RawDoubleHyperparameter      *DoubleHyperparameterV0      `union:"type,double" json:"-"`
	RawLogHyperparameter         *LogHyperparameterV0         `union:"type,log" json:"-"`
	RawCategoricalHyperparameter *CategoricalHyperparameterV0 `union:"type,categorical" json:"-"`
}

// Merge prevents recursive merging of hyperparameters.
func (h HyperparameterV0) Merge(other interface{}) interface{} {
	// Never merge partial hyperparameters.
	return h
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (h *HyperparameterV0) UnmarshalJSON(data []byte) error {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	if _, ok := parsed.(map[string]interface{}); ok {
		return union.Unmarshal(data, h)
	}
	h.RawConstHyperparameter = &ConstHyperparameterV0{RawVal: parsed}
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (h HyperparameterV0) MarshalJSON() ([]byte, error) {
	return union.MarshalEx(h, true)
}

// Each applies the function to each hyperparameter in string order of the name.
func (h HyperparametersV0) Each(f func(name string, param HyperparameterV0)) {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, h[k])
	}
}

//go:generate ../gen.sh
// ConstHyperparameterV0 is a constant.
type ConstHyperparameterV0 struct {
	RawVal interface{} `json:"val"`
}

//go:generate ../gen.sh
// IntHyperparameterV0 is an interval of ints.
type IntHyperparameterV0 struct {
	RawMinval int  `json:"minval"`
	RawMaxval int  `json:"maxval"`
	RawCount  *int `json:"count,omitempty"`
}

//go:generate ../gen.sh
// DoubleHyperparameterV0 is an interval of float64s.
type DoubleHyperparameterV0 struct {
	RawMinval float64 `json:"minval"`
	RawMaxval float64 `json:"maxval"`
	RawCount  *int    `json:"count,omitempty"`
}

//go:generate ../gen.sh
// LogHyperparameterV0 is a log-uniformly distributed interval of float64s.
type LogHyperparameterV0 struct {
	// Minimum value is `base ^ minval`.
	RawMinval float64 `json:"minval"`
	// Maximum value is `base ^ maxval`.
	RawMaxval float64 `json:"maxval"`
	RawBase   float64 `json:"base"`
	RawCount  *int    `json:"count,omitempty"`
}

//go:generate ../gen.sh
// CategoricalHyperparameterV0 is a collection of values (levels) of the category.
type CategoricalHyperparameterV0 struct {
	RawVals []interface{} `json:"vals"`
}
