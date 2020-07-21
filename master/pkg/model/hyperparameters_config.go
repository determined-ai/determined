package model

import (
	"encoding/json"
	"sort"

	"github.com/determined-ai/determined/master/pkg/check"
	"github.com/determined-ai/determined/master/pkg/union"
)

// GlobalBatchSize is the name of the hyperparameter for global_batch_size.
const GlobalBatchSize = "global_batch_size"

// Hyperparameters holds a mapping from hyperparameter name to its configuration.
type Hyperparameters map[string]Hyperparameter

// Each applies the function to each hyperparameter in string order of the name.
func (h Hyperparameters) Each(f func(name string, param Hyperparameter)) {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		f(k, h[k])
	}
}

// MaxGlobalBatchSize returns global_batch_size if it is a const or the max it can be if it is
// variable.
func (h Hyperparameters) MaxGlobalBatchSize() int {
	switch hp := h[GlobalBatchSize]; {
	case hp.ConstHyperparameter != nil:
		return int(hp.ConstHyperparameter.Val.(float64))
	case hp.IntHyperparameter != nil:
		return hp.IntHyperparameter.Maxval
	default:
		panic("didn't find global_batch_size")
	}
}

// Hyperparameter is a sum type for hyperparameters.
type Hyperparameter struct {
	ConstHyperparameter       *ConstHyperparameter       `union:"type,const" json:"-"`
	IntHyperparameter         *IntHyperparameter         `union:"type,int" json:"-"`
	DoubleHyperparameter      *DoubleHyperparameter      `union:"type,double" json:"-"`
	LogHyperparameter         *LogHyperparameter         `union:"type,log" json:"-"`
	CategoricalHyperparameter *CategoricalHyperparameter `union:"type,categorical" json:"-"`
}

// MarshalJSON implements the json.Marshaler interface.
func (h Hyperparameter) MarshalJSON() ([]byte, error) {
	return union.Marshal(h)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (h *Hyperparameter) UnmarshalJSON(data []byte) error {
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

// ConstHyperparameter is a constant.
type ConstHyperparameter struct {
	Val interface{} `json:"val"`
}

// IntHyperparameter is an interval of ints.
type IntHyperparameter struct {
	Minval int  `json:"minval"`
	Maxval int  `json:"maxval"`
	Count  *int `json:"count"`
}

// Validate implements the check.Validatable interface.
func (i IntHyperparameter) Validate() []error {
	return []error{
		check.GreaterThan(i.Maxval, i.Minval, "minval is greater than maxval"),
		check.GreaterThan(i.Count, 0, "count must be >= 0"),
	}
}

// DoubleHyperparameter is an interval of float64s.
type DoubleHyperparameter struct {
	Minval float64 `json:"minval"`
	Maxval float64 `json:"maxval"`
	Count  *int    `json:"count"`
}

// Validate implements the check.Validatable interface.
func (d DoubleHyperparameter) Validate() []error {
	return []error{
		check.GreaterThan(d.Maxval, d.Minval, "minval is greater than maxval"),
		check.GreaterThan(d.Count, 0, "count must be >= 0"),
	}
}

// LogHyperparameter is a log-uniformly distributed interval of float64s.
type LogHyperparameter struct {
	// Minimum value is `base ^ minval`.
	Minval float64 `json:"minval"`
	// Maximum value is `base ^ maxval`.
	Maxval float64 `json:"maxval"`
	Base   float64 `json:"base"`
	Count  *int    `json:"count"`
}

// Validate implements the check.Validatable interface.
func (h *LogHyperparameter) Validate() []error {
	return []error{
		check.GreaterThan(h.Maxval, h.Minval, "minval is greater than maxval"),
		check.GreaterThan(h.Base, 0.0, "base must be >= 0"),
		check.GreaterThan(h.Count, 0, "count must be >= 0"),
	}
}

// CategoricalHyperparameter is a collection of values (levels) of the category.
type CategoricalHyperparameter struct {
	Vals []interface{} `json:"vals"`
}

// Validate implements the check.Validatable interface.
func (h *CategoricalHyperparameter) Validate() []error {
	return []error{
		check.GreaterThan(len(h.Vals), 0, "must have at least one category"),
	}
}
