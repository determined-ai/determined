package apiutils

import (
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
	
	paths := m.Paths()
	fields := make(map[string]bool, len(paths))
	for _, f := paths {
		fields[f] = true
	}

	m.IsValid()

	return FieldMask{
		m: m,
		fieldSet: fields,
	}
}

// FieldInSet answers whether the passed-in field is in the FieldMask. FieldInSet respects the 
// FieldMask convention of treating empty FieldMasks as containing every field.
func (f *FieldMask) FieldInSet(field string) bool {
	if len(f.fieldSet) == 0 {
		return true
	}

	return f.fieldSet[field]
}
