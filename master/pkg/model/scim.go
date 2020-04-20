package model

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// The following are constants for the SCIM v2.0 protocol:
//
// https://tools.ietf.org/html/rfc7644
const (
	scimUserType   = "User"
	scimUserSchema = "urn:ietf:params:scim:schemas:core:2.0:User"

	scimGroupType   = "Group"
	scimGroupSchema = "urn:ietf:params:scim:schemas:core:2.0:Group"

	scimListSchema = "urn:ietf:params:scim:api:messages:2.0:ListResponse"

	scimErrorSchema = "urn:ietf:params:scim:api:messages:2.0:Error"
)

// validateSchemas verifies that the given schema is the only element of the
// data array as encoded as a raw JSON byte string.
func validateSchemas(schema string, data []byte) error {
	var schemas []string
	if err := json.Unmarshal(data, &schemas); err != nil {
		return errors.WithStack(err)
	}

	if n := len(schemas); n == 0 {
		return errors.New("no schemas found")
	} else if n > 1 {
		return errors.New("more than one schema found")
	}

	if schemas[0] != schema {
		return errors.Errorf("unknown schema, expecting %s", schema)
	}

	return nil
}

// validateString verifies that the given string matches the data string as
// encoded as a raw JSON byte string.
func validateString(s string, data []byte) error {
	var d string
	if err := json.Unmarshal(data, &d); err != nil {
		return errors.WithStack(err)
	}
	if d != s {
		return errors.Errorf("unknown string, expecting %s", s)
	}

	return nil
}

// scanJSON unmarshals a source value into the destination.
func scanJSON(src, dst interface{}) error {
	switch src := src.(type) {
	case []byte:
		return errors.WithStack(json.Unmarshal(src, dst))
	default:
		return errors.Errorf("unknown type %T", src)
	}
}

// SCIMErrorSchemas is the constant schemas field for errors.
type SCIMErrorSchemas struct{}

// MarshalJSON implements json.Marshaler.
func (s SCIMErrorSchemas) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{scimErrorSchema})
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SCIMErrorSchemas) UnmarshalJSON(data []byte) error {
	return validateSchemas(scimErrorSchema, data)
}

// SCIMError is an error in SCIM.
type SCIMError struct {
	Detail   string           `json:"detail,omitempty"`
	Status   int              `json:"status"`
	SCIMType string           `json:"scimType,omitempty"`
	Schemas  SCIMErrorSchemas `json:"schemas"`
}

// SCIMListSchemas is the constant schemas field for lists.
type SCIMListSchemas struct{}

// MarshalJSON implements json.Marshaler.
func (s SCIMListSchemas) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{scimListSchema})
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *SCIMListSchemas) UnmarshalJSON(data []byte) error {
	return validateSchemas(scimListSchema, data)
}
