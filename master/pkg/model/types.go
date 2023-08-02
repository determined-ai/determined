package model

import (
	"database/sql/driver"
	"encoding/json"
	"math"

	"github.com/pkg/errors"
)

// JSONObj is a JSON object that converts to a []byte in SQL queries.
type JSONObj map[string]interface{}

// Value marshals a []byte.
func (j JSONObj) Value() (driver.Value, error) {
	bytes, err := json.Marshal(j)
	if err != nil {
		return nil, errors.Wrap(err, "error marshaling JSONObj")
	}
	return bytes, nil
}

// Scan unmarshals JSON in []byte to map[string]interface{}.
func (j *JSONObj) Scan(src interface{}) error {
	if src == nil {
		*j = nil
		return nil
	}
	bytes, ok := src.([]byte)
	if !ok {
		return errors.Errorf("unable to convert to []byte: %v", src)
	}
	obj := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &obj); err != nil {
		return errors.Wrapf(err, "unable to unmarshal JSONObj: %v", src)
	}
	*j = JSONObj(obj)
	return nil
}

// ExtendedFloat64 handles serializing floats to JSON, including special cases for infinite values.
type ExtendedFloat64 float64

// MarshalJSON implements the json.Marshaler interface.
func (f ExtendedFloat64) MarshalJSON() ([]byte, error) {
	switch float64(f) {
	case math.Inf(1):
		return []byte(`"Infinity"`), nil
	case math.Inf(-1):
		return []byte(`"-Infinity"`), nil
	}
	return json.Marshal(float64(f))
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (f *ExtendedFloat64) UnmarshalJSON(data []byte) error {
	var inner float64
	switch string(data) {
	case `"Infinity"`:
		inner = math.Inf(1)
	case `"-Infinity"`:
		inner = math.Inf(-1)
	default:
		if err := json.Unmarshal(data, &inner); err != nil {
			return err
		}
	}
	*f = ExtendedFloat64(inner)
	return nil
}
