package model

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
)

// Duration is a JSON (un)marshallable version of time.Duration.
type Duration time.Duration

// MarshalJSON implements the json.Marshaler interface.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case string:
		tmp, err := time.ParseDuration(value)
		if err != nil {
			return errors.Wrap(err, "error parsing duration")
		}
		*d = Duration(tmp)
		return nil
	default:
		return errors.Errorf("invalid duration: %s", b)
	}
}
