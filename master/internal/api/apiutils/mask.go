package apiutils

import (
	"strings"

	field_mask "google.golang.org/genproto/protobuf/field_mask"
)

// FieldMask is a utility type for efficiently interacting with protobuf FieldMasks in a compliant
// way.
type FieldMask struct {
	m        *field_mask.FieldMask
	fieldSet map[string]bool
}

// NewFieldMask initializes a FieldMask object from a protobuf FieldMask pointer. A passed-in nil
// or a FieldMask with zero paths will result in an empty FieldMask, which is considered to contain
// every field.
func NewFieldMask(m *field_mask.FieldMask) FieldMask {
	if m == nil {
		return FieldMask{}
	}

	m.Normalize()

	paths := m.GetPaths()
	fields := make(map[string]bool, len(paths))
	for _, f := range paths {
		fields[f] = true
	}

	return FieldMask{
		m:        m,
		fieldSet: fields,
	}
}

// FieldInSet answers whether the passed-in field is in the FieldMask. FieldInSet respects the
// FieldMask convention of treating empty FieldMasks as containing every field.
func (f *FieldMask) FieldInSet(field string) bool {
	if len(f.fieldSet) == 0 {
		return true
	}

	if f.fieldSet[field] {
		return true
	}

	// If the fieldmask contains a.b and a user inquires about field a.b.c, the answer is yes.
	fields := strings.Split(field, ".")
	if len(fields) > 1 {
		// Continually re-slice the slice of fields to check if an ancestor of field was specified.
		for fields = fields[:len(fields)-1]; len(fields) > 0; fields = fields[:len(fields)-1] {
			path := strings.Join(fields, ".")
			if f.fieldSet[path] {
				return true
			}
		}
	}

	return false
}
