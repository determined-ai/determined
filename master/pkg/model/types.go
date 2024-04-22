package model

import (
	"database/sql/driver"
	"encoding/json"
	"math"

	"github.com/google/uuid"
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

// UUID is a UUID that converts to a nullable string in SQL queries.
type UUID struct {
	UUID  uuid.UUID
	Valid bool
}

// NewUUID creates a new, non-null and random UUID.
func NewUUID() UUID {
	return UUID{
		UUID:  uuid.New(),
		Valid: true,
	}
}

// ParseUUID initializes a non-null UUID from a string. It returns an error if
// the string does not follow the format of a UUID.
func ParseUUID(s string) (UUID, error) {
	x, err := uuid.Parse(s)
	if err != nil {
		return UUID{}, errors.WithStack(err)
	}

	return UUID{
		UUID:  x,
		Valid: true,
	}, nil
}

// String returns the string representation of the UUID. If this UUID is null,
// return the empty string.
func (u UUID) String() string {
	if !u.Valid {
		return ""
	}
	return u.UUID.String()
}

// MarshalJSON implements the json.Marshaler interface.
func (u UUID) MarshalJSON() ([]byte, error) {
	bs, err := json.Marshal(u.String())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return bs, nil
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (u *UUID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return errors.WithStack(err)
	}

	x, err := ParseUUID(s)
	if err != nil {
		return err
	}

	*u = x

	return nil
}

// Value implements the sql.Driver interface.
func (u UUID) Value() (driver.Value, error) {
	if !u.Valid {
		return nil, nil
	}
	return u.String(), nil
}

// Scan implements the sql.Scanner interface.
func (u *UUID) Scan(value interface{}) error {
	if value == nil {
		u.UUID = uuid.UUID{}
		u.Valid = false
		return nil
	}

	var x uuid.UUID
	var err error

	switch v := value.(type) {
	case string:
		x, err = uuid.Parse(v)
	case []byte:
		x, err = uuid.Parse(string(v))
	default:
		return errors.Errorf("unknown type %T", v)
	}

	if err != nil {
		return errors.WithStack(err)
	}

	u.UUID = x
	u.Valid = true
	return nil
}
