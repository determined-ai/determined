package expconf

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/determined-ai/determined/master/pkg/union"
)

// GlobalBatchSize is the name of the hyperparameter for global_batch_size.
const GlobalBatchSize = "global_batch_size"

//go:generate ../gen.sh
// HyperparametersV0 is a versioned hyperparameters config.
type HyperparametersV0 map[string]interface{}

// FlattenHPs returns a flat dictionary with keys representing nested structure.
// For example, {"optimizer": {"learning_rate": 0.01}} will be flattened to
// {"optimizer.learning_rate": 0.01}.
func FlattenHPs(h HyperparametersV0) HyperparametersV0 {
	flatHPs := make(HyperparametersV0)
	flattenHPs(h, "", flatHPs)
	return flatHPs
}

func flattenHPs(h HyperparametersV0, prefix string, target HyperparametersV0) {
	for key, element := range h {
		switch tHP := element.(type) {
		case HyperparametersV0:
			flattenHPs(tHP, prefix+key+".", target)
		default:
			target[prefix+key] = element
		}
	}
}

// UnflattenHPs undos the FlattenHPs function to restore the nested structure.
func UnflattenHPs(h HyperparametersV0) HyperparametersV0 {
	result := make(HyperparametersV0)
	for key, element := range h {
		nesting := strings.Split(key, ".")
		hPointer := result
		if len(nesting) > 1 {
			for i := 0; i < len(nesting)-1; i++ {
				k := nesting[i]
				if _, ok := hPointer[k]; !ok {
					hPointer[k] = make(HyperparametersV0)
				}
				hPointer = hPointer[k].(HyperparametersV0)
			}
		}
		hPointer[nesting[len(nesting)-1]] = element
	}
	return result
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (h *HyperparametersV0) UnmarshalJSON(data []byte) error {
	if *h == nil {
		*h = make(map[string]interface{})
	}
	var parsed interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	if _, ok := parsed.(map[string]interface{}); ok {
		for key, element := range parsed.(map[string]interface{}) {
			if _, isDict := element.(map[string]interface{}); isDict {
				hpBytes, _ := json.Marshal(element)
				var hp HyperparameterV0
				err := json.Unmarshal(hpBytes, &hp)
				if err != nil {
					var hps HyperparametersV0
					err := hps.UnmarshalJSON(hpBytes)
					if err != nil {
						return err
					}
					(*h)[key] = hps
				} else {
					(*h)[key] = hp
				}
			} else {
				var hp HyperparameterV0
				hp.RawConstHyperparameter = &ConstHyperparameterV0{RawVal: element}
				(*h)[key] = hp
			}
		}
	}
	return nil
}

func hpsToMap(hps HyperparametersV0) (map[string]interface{}, error) {
	output := make(map[string]interface{})
	var err error
	for key, element := range hps {
		switch hp := element.(type) {
		case HyperparameterV0:
			hpBytes, _ := hp.MarshalJSON()
			var unionHP map[string]interface{}
			err = json.Unmarshal(hpBytes, &unionHP)
			if err != nil {
				return nil, err
			}
			output[key] = unionHP
		case HyperparametersV0:
			output[key], err = hpsToMap(hp)
			if err != nil {
				return nil, err
			}
		}
	}
	return output, nil
}

// MarshalJSON implements the json.Marshaler interface.
func (h HyperparametersV0) MarshalJSON() ([]byte, error) {
	if h == nil {
		return []byte("null"), nil
	}
	hps, err := hpsToMap(h)
	if err != nil {
		return []byte("null"), err
	}
	return json.Marshal(hps)
}

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
		switch hp := h[k].(type) {
		case HyperparameterV0:
			f(k, hp)
		case HyperparametersV0:
			hp.Each(f)
		}
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
