package union

import (
	"reflect"

	"github.com/pkg/errors"
)

type (
	// unionTypes maps each type key to the set of possible type values.
	unionTypes map[string]unionType
	// unionType maps each union type value to all the union fields.
	unionType map[string]unionField
	// unionField stores the struct field and its index in the struct.
	unionField struct {
		index int
		field reflect.StructField
	}
)

// parseUnionTypes returns union type information for the provided type.
func parseUnionTypes(v reflect.Type) (unionTypes, error) {
	types := make(unionTypes)
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if tagValue, ok := field.Tag.Lookup(unionTag); ok {
			// Parse the union type and value for the specific field.
			key, value, err := parseUnionStructTag(tagValue)
			if err != nil {
				return nil, err
			}

			// If this is the first union type for this key, initialize the map.
			if _, ok := types[key]; !ok {
				types[key] = make(unionType)
			}

			if field.Type.Kind() != reflect.Ptr {
				return nil, errors.Errorf(
					"%s expected to a pointer type: found %s", field.Name, field.Type.Kind())
			}
			types[key][value] = unionField{index: i, field: field}
		}
	}
	return types, nil
}
