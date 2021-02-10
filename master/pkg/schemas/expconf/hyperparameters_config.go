package expconf

import (
	"encoding/json"
	"sort"

	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/union"
)

// GlobalBatchSize is the name of the hyperparameter for global_batch_size.
const GlobalBatchSize = "global_batch_size"

// HyperparametersV0 is a versioned hyperparameters config.
type HyperparametersV0 map[string]HyperparameterV0

// HyperparameterV0 is a sum type for hyperparameters.
type HyperparameterV0 struct {
	ConstHyperparameter       *ConstHyperparameter       `union:"type,const" json:"-"`
	IntHyperparameter         *IntHyperparameter         `union:"type,int" json:"-"`
	DoubleHyperparameter      *DoubleHyperparameter      `union:"type,double" json:"-"`
	LogHyperparameter         *LogHyperparameter         `union:"type,log" json:"-"`
	CategoricalHyperparameter *CategoricalHyperparameter `union:"type,categorical" json:"-"`
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
	h.ConstHyperparameter = &ConstHyperparameter{Val: parsed}
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (h HyperparameterV0) MarshalJSON() ([]byte, error) {
	return union.MarshalEx(h, true)
}

// DefaultSource implements the Defaultable interface.
func (h HyperparameterV0) DefaultSource() interface{} {
	return schemas.UnionDefaultSchema(h)
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

// ConstHyperparameterV0 is a constant.
type ConstHyperparameterV0 struct {
	Val interface{} `json:"val"`
}

// IntHyperparameterV0 is an interval of ints.
type IntHyperparameterV0 struct {
	Minval int  `json:"minval"`
	Maxval int  `json:"maxval"`
	Count  *int `json:"count,omitempty"`
}

// DoubleHyperparameterV0 is an interval of float64s.
type DoubleHyperparameterV0 struct {
	Minval float64 `json:"minval"`
	Maxval float64 `json:"maxval"`
	Count  *int    `json:"count,omitempty"`
}

// LogHyperparameterV0 is a log-uniformly distributed interval of float64s.
type LogHyperparameterV0 struct {
	// Minimum value is `base ^ minval`.
	Minval float64 `json:"minval"`
	// Maximum value is `base ^ maxval`.
	Maxval float64 `json:"maxval"`
	Base   float64 `json:"base"`
	Count  *int    `json:"count,omitempty"`
}

// CategoricalHyperparameterV0 is a collection of values (levels) of the category.
type CategoricalHyperparameterV0 struct {
	Vals []interface{} `json:"vals"`
}
