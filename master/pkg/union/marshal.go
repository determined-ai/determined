package union

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// MarshalEx allows for configurable error handling.
func MarshalEx(v interface{}, allowEmptyUnion bool) ([]byte, error) {
	data := make(map[string]interface{})
	value := reflect.ValueOf(v)
	valueType := reflect.TypeOf(v)

	unionTypes, err := parseUnionTypes(valueType)
	if err != nil {
		return nil, err
	}

	// Set all union type fields in the map.
	for key, fields := range unionTypes {
		unionDefined := false
		for name, field := range fields {
			// Only marshal union types that are defined.
			if value.Field(field.index).IsNil() {
				continue
			}

			// Only one union type can be marshaled into the JSON struct.
			if unionDefined {
				return nil, errors.Errorf(
					"%s defines a union field when one is already defined", key)
			}

			// Marshal the union type to a map and add each field to the parent map.
			nested, err := marshalToMap(value.Field(field.index).Interface())
			if err != nil {
				return nil, err
			}
			// TODO: Check if union types have interfering field names.
			for nestedKey, nestedValue := range nested {
				data[nestedKey] = nestedValue
			}

			// Set the union type key to the union type name.
			data[key] = name
			unionDefined = true
		}

		// At least one union type must be defined.
		if !unionDefined && !allowEmptyUnion {
			return nil, errors.Errorf("no union field defined: %s", key)
		}
	}

	// Marshal all the fields in the base struct.
	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		switch jsonTagValue, ok := field.Tag.Lookup("json"); {
		case jsonTagValue == "-":
			continue
		case !ok:
			jsonTagValue = field.Name
			fallthrough
		default:
			if strings.Contains(jsonTagValue, ",") {
				return nil, errors.New(
					"advanced json tag features not support in union type marshaling")
			}
			data[jsonTagValue] = value.Field(i).Interface()
		}
	}

	return json.Marshal(data)
}

// Marshal marshals the provided union type into a JSON byte array.
func Marshal(v interface{}) ([]byte, error) {
	return MarshalEx(v, false)
}

// marshalToMap returns a map representation of the provided interface.
func marshalToMap(v interface{}) (map[string]interface{}, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	data := make(map[string]interface{})
	err = json.Unmarshal(bytes, &data)
	return data, err
}
