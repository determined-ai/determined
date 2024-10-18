package union

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// Unmarshal unmarshals the provided union type from a JSON byte array.
func Unmarshal(data []byte, v interface{}) error {
	value := reflect.ValueOf(v)
	expectedFields := make(map[string]bool)
	unionTypes, err := parseUnionTypes(value.Type().Elem())
	if err != nil {
		return err
	}

	for key, fields := range unionTypes {
		expectedFields[key] = true
		expectedValue, ok, err := getTagValue(data, key)
		if err != nil {
			return err
		} else if !ok {
			continue
		}
		field, ok := fields[expectedValue]
		if !ok {
			return errors.Errorf("unexpected %s: %s", key, expectedValue)
		}

		if fieldVal := value.Elem().Field(field.index); !fieldVal.IsNil() {
			if err := json.Unmarshal(data, fieldVal.Interface()); err != nil {
				return err
			}
		} else {
			nested := reflect.New(field.field.Type.Elem())
			if err := json.Unmarshal(data, nested.Interface()); err != nil {
				return err
			}
			fieldVal.Set(nested)
		}

		for _, other := range fields {
			if other.index == field.index {
				continue
			}
			value.Elem().Field(other.index).Set(reflect.Zero(other.field.Type))
		}

		for k := range parseFields(field.field.Type.Elem()) {
			expectedFields[k] = true
		}
	}
	for k := range parseFields(value.Type().Elem()) {
		expectedFields[k] = true
	}
	return checkFields(expectedFields, data)
}

func parseFields(elem reflect.Type) map[string]bool {
	fields := make(map[string]bool)
	for i := 0; i < elem.NumField(); i++ {
		field := elem.Field(i)
		switch jsonTagValue, ok := field.Tag.Lookup("json"); {
		case jsonTagValue == "-":
			continue
		case !ok:
			jsonTagValue = field.Name
			fallthrough
		default:
			if strings.Contains(jsonTagValue, ",") {
				fields[strings.Split(jsonTagValue, ",")[0]] = true
			} else {
				fields[jsonTagValue] = true
			}
		}
	}
	return fields
}

func checkFields(fields map[string]bool, bytes []byte) error {
	data := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &data); err != nil {
		return err
	}
	for key := range data {
		if _, ok := fields[key]; !ok {
			return errors.Errorf("json: unknown field \"%s\"", key)
		}
	}
	return nil
}
