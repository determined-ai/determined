package expconf

import (
	"encoding/json"
	"sort"

	"github.com/determined-ai/determined/master/pkg/union"
)

// HyperparametersV0 is a versioned hyperparameters config.
//
//go:generate ../gen.sh
type HyperparametersV0 map[string]HyperparameterV0

// FlattenHPs returns a flat dictionary with keys representing nested structure.
// For example, {"optimizer": {"learning_rate": 0.01}} will be flattened to
// {"optimizer.learning_rate": 0.01}.
func FlattenHPs(h HyperparametersV0) HyperparametersV0 {
	flatHPs := make(HyperparametersV0)
	for key, val := range h {
		if val.RawNestedHyperparameter != nil {
			flattenNestedHP(val, key+".", &flatHPs)
		} else {
			flatHPs[key] = val
		}
	}
	return flatHPs
}

func flattenNestedHP(h HyperparameterV0, prefix string, target *HyperparametersV0) {
	for key, val := range *h.RawNestedHyperparameter {
		if val.RawNestedHyperparameter != nil {
			flattenNestedHP(val, prefix+key+".", target)
		} else {
			(*target)[prefix+key] = val
		}
	}
}

// HyperparameterV0 is a sum type for hyperparameters.
//
//go:generate ../gen.sh
type HyperparameterV0 struct {
	RawConstHyperparameter       *ConstHyperparameterV0       `union:"type,const" json:"-"`
	RawIntHyperparameter         *IntHyperparameterV0         `union:"type,int" json:"-"`
	RawDoubleHyperparameter      *DoubleHyperparameterV0      `union:"type,double" json:"-"`
	RawLogHyperparameter         *LogHyperparameterV0         `union:"type,log" json:"-"`
	RawCategoricalHyperparameter *CategoricalHyperparameterV0 `union:"type,categorical" json:"-"`
	// RawNestedHyperparameter is added as a union type to more closely reflect the underlying
	// schema definition. Doing so also means that we can detect a nested hyperparameter from
	// a call to the automatically generated HyperparameterV0.GetUnionMember function.
	//
	// However, this type does not actually go through union marshaling and unmarshaling logic
	// so that we can support implicit nesting as follows:
	// optimizer:
	//   learning_rate: 0.01
	//   momentum: 0.9
	// This means that we do NOT support explicit nesting as we would normally with a union
	// on type:
	// optimizer:
	//   type: object
	//   vals:
	//     learning_rate: 0.01
	//     momentum: 0.9
	// The former is more user friendly so we will escape the union unmarshaling logic
	// in its favor. We can add the later behavior in the future if needed.
	RawNestedHyperparameter *map[string]HyperparameterV0 `union:"type,object" json:"-"`
}

// Merge prevents recursive merging of hyperparameters unless h
// is a nested hyperparameter.  When h is a nested hyperparameter,
// we merge fields of h and other into a single map.
// A new HyperparameterV0 instance is returned.
func (h HyperparameterV0) Merge(other HyperparameterV0) HyperparameterV0 {
	// Only merge nested hyperparameters.
	if h.RawNestedHyperparameter != nil && other.RawNestedHyperparameter != nil {
		newNestedHP := make(map[string]HyperparameterV0)
		target := *h.RawNestedHyperparameter
		source := *other.RawNestedHyperparameter
		for key, val := range target {
			if sourceVal, inSource := source[key]; inSource {
				newNestedHP[key] = val.Merge(sourceVal)
			} else {
				newNestedHP[key] = val
			}
		}
		for key, val := range source {
			if _, inTarget := target[key]; !inTarget {
				newNestedHP[key] = val
			}
		}
		return HyperparameterV0{RawNestedHyperparameter: &newNestedHP}
	}
	return h
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (h *HyperparameterV0) UnmarshalJSON(data []byte) error {
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	if parsedMap, ok := parsed.(map[string]interface{}); ok {
		_, hasType := parsedMap["type"]
		// If "type" not in map, then we have a nested hp, which
		// we will unmarshal into map[string]HyperparameterV0 instead
		// of using the union unmarshaling logic.
		if !hasType {
			return json.Unmarshal(data, &h.RawNestedHyperparameter)
		}
		return union.Unmarshal(data, h)
	}
	h.RawConstHyperparameter = &ConstHyperparameterV0{RawVal: parsed}
	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (h HyperparameterV0) MarshalJSON() ([]byte, error) {
	// Intercept union marshaling logic for nested hps and directly
	// marshal the underlying map[string]HyperparameterV0.
	if h.RawNestedHyperparameter != nil {
		return json.Marshal(*h.RawNestedHyperparameter)
	}
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

// ConstHyperparameterV0 is a constant.
//
//go:generate ../gen.sh
type ConstHyperparameterV0 struct {
	RawVal interface{} `json:"val"`
}

// IntHyperparameterV0 is an interval of ints.
//
//go:generate ../gen.sh
type IntHyperparameterV0 struct {
	RawMinval int  `json:"minval"`
	RawMaxval int  `json:"maxval"`
	RawCount  *int `json:"count,omitempty"`
}

// DoubleHyperparameterV0 is an interval of float64s.
//
//go:generate ../gen.sh
type DoubleHyperparameterV0 struct {
	RawMinval float64 `json:"minval"`
	RawMaxval float64 `json:"maxval"`
	RawCount  *int    `json:"count,omitempty"`
}

// LogHyperparameterV0 is a log-uniformly distributed interval of float64s.
//
//go:generate ../gen.sh
type LogHyperparameterV0 struct {
	// Minimum value is `base ^ minval`.
	RawMinval float64 `json:"minval"`
	// Maximum value is `base ^ maxval`.
	RawMaxval float64 `json:"maxval"`
	RawBase   float64 `json:"base"`
	RawCount  *int    `json:"count,omitempty"`
}

// CategoricalHyperparameterV0 is a collection of values (levels) of the category.
//
//go:generate ../gen.sh
type CategoricalHyperparameterV0 struct {
	RawVals []interface{} `json:"vals"`
}
