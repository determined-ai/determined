package model

import "encoding/json"

const (
	patchSchema = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
)

// A PatchOperation is a RFC 6902 JSON Patch.
//
// https://tools.ietf.org/html/rfc6902
type PatchOperation struct {
	// Op is one of add, remove, replace, move, copy, test.
	Op string `json:"op"`

	// Path is the field to update.
	Path string `json:"path"`

	// Value is the new value.
	Value json.RawMessage `json:"value"`
}

// A PatchRequest is a SCIM patch request.
type PatchRequest struct {
	Schemas    PatchSchemas     `json:"schemas"`
	Operations []PatchOperation `json:"operations"`
}

// PatchSchemas is a constant schemas field for a patch.
type PatchSchemas struct{}

// MarshalJSON implements json.Marshaler.
func (s PatchSchemas) MarshalJSON() ([]byte, error) {
	return json.Marshal([]string{
		patchSchema,
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (s *PatchSchemas) UnmarshalJSON(data []byte) error {
	return validateSchemas(patchSchema, data)
}
