package model

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/pkg/errors"
)

// JSONObj is a JSON object that converts to a []byte in SQL queries.
type JSONObj map[string]interface{}

// JSONObjFromMapStringInt64 converts map[string]int64 to a JSONObj.
func JSONObjFromMapStringInt64(m map[string]int64) JSONObj {
	r := make(JSONObj)
	for k, v := range m {
		r[k] = v
	}
	return r
}

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

// RawString is a string that encodes as a byte array when read or written to a
// database yet is represented as a string otherwise.
//
// Postgres does not allow zero bytes ('\x00') in char fields. The UTF-8
// encoding of the Unicode code point NUL (U+0000) is the zero byte '\x00'.
// Thus, Postgres rejects valid UTF-8 strings. RawString helps work around this
// problem by transparently saving UTF-8 strings as raw bytes (bytea) in the
// database but otherwise behaving like a string (e.g., when marshaled as JSON).
type RawString string

// Value implements the driver.Valuer interface.
func (s RawString) Value() (driver.Value, error) {
	return []byte(s), nil
}

// Scan implements the sql.Scanner interface.
func (s *RawString) Scan(src interface{}) error {
	switch src := src.(type) {
	case []byte:
		*s = RawString(src)
	default:
		return errors.Errorf("unexpected type: %T", src)
	}
	return nil
}
