package run

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/pkg/model"
)

const (
	integerType   = "integer"
	floatType     = "float"
	booleanType   = "boolean"
	stringType    = "string"
	timestampType = "timestamp"
	// MaxMetadataValueStringLength is the maximum length of a metadata value.
	MaxMetadataValueStringLength = 5000
	// MaxMetadataKeyLength is the maximum length of a metadata key.
	MaxMetadataKeyLength = 50
	// MaxMetadataArrayLength is the maximum length of a metadata array.
	MaxMetadataArrayLength = 100
	// MaxMetadataDepth is the maximum depth of nested metadata.
	MaxMetadataDepth = 10
	// ExcludedMetadataCharactersPattern is the pattern of characters that are not allowed in metadata values.
	ExcludedMetadataCharactersPattern = "[\\$\\.\\[\\]]"
)

// FlattenRunMetadata flattens a nested map of run metadata into a list of RunMetadataIndex entries.
func FlattenRunMetadata(data map[string]interface{}) (flatKeys []model.RunMetadataIndex, keyCount int, err error) {
	if len(data) == 0 {
		return nil, 0, fmt.Errorf("metadata is empty")
	}
	return flattenRunMetadata(data, "", 0)
}

// parseMetadataValueType converts a value to a string and returns the type of the value.
func parseMetadataValueType(value interface{}) (string, string) {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v), integerType
	case float64:
		if v == float64(int(v)) {
			return strconv.Itoa(int(v)), integerType
		}
		return strconv.FormatFloat(v, 'f', -1, 64), floatType
	case bool:
		return strconv.FormatBool(v), booleanType
	case string:
		if _, err := time.Parse(time.RFC3339Nano, v); err == nil {
			return v, "timestamp"
		} else if timestamp, err := time.Parse(time.RFC3339, v); err == nil {
			return timestamp.UTC().Format(time.RFC3339Nano), timestampType
		} else if timestamp, err := time.Parse("2006-01-02", v); err == nil {
			return timestamp.UTC().Format(time.RFC3339Nano), timestampType
		} else if timestamp, err := time.Parse("2006-01", v); err == nil {
			return timestamp.UTC().Format(time.RFC3339Nano), timestampType
		}
		return v, stringType
	default:
		return fmt.Sprintf("%v", v), reflect.TypeOf(value).String()
	}
}

func flattenRunMetadata(data map[string]interface{}, prefix string, depth int) ([]model.RunMetadataIndex, int, error) {
	if depth > MaxMetadataDepth {
		return nil, 0, fmt.Errorf("metadata exceeds maximum nesting depth of %d", MaxMetadataDepth)
	}
	flattenedKeys := make([]model.RunMetadataIndex, 0)
	numKeys := 0
	for key, value := range data {
		if len(key) > MaxMetadataKeyLength {
			return nil, 0, fmt.Errorf("metadata key exceeds maximum length of %d characters", MaxMetadataKeyLength)
		}
		// count the key we're currently processing
		numKeys++
		newKey := fmt.Sprintf("%s%s", prefix, key)
		switch typedVal := value.(type) {
		// If the value is a map, recursively flatten it.
		case map[string]interface{}:
			// these we should add
			flatKeys, nestedKeyCount, err := flattenRunMetadata(typedVal, newKey+".", depth+1)
			if err != nil {
				return nil, 0, err
			}
			flattenedKeys = append(flattenedKeys, flatKeys...)
			numKeys += nestedKeyCount
		// If the value is a slice, iterate over it and recursively flatten each element.
		case []interface{}:
			// TODO (corban): we only need to add to this if there is map element(s) in the array
			if len(typedVal) > MaxMetadataArrayLength {
				return nil, 0, fmt.Errorf("metadata array exceeds maximum length of %d/%d elements",
					len(typedVal),
					MaxMetadataArrayLength,
				)
			}

			for _, v := range typedVal {
				switch typedElem := v.(type) {
				case map[string]interface{}:
					// TODO (corban): this is where we'd do it
					flatKeys, nestedKeyCount, err := flattenRunMetadata(typedElem, newKey+".", depth+1)
					if err != nil {
						return nil, 0, err
					}
					flattenedKeys = append(flattenedKeys, flatKeys...)
					numKeys += nestedKeyCount
				case []interface{}:
					if len(typedElem) > MaxMetadataArrayLength {
						return nil, 0, fmt.Errorf("metadata array exceeds maximum length of %d/%d elements",
							len(typedElem),
							MaxMetadataArrayLength,
						)
					}

					flatKeys, nestedKeyCount, err := flattenRunMetadata(map[string]interface{}{newKey: typedElem}, "", depth+1)
					if err != nil {
						return nil, 0, err
					}
					// if there are nested keys, we need to add them to the key count
					if nestedKeyCount > 1 {
						numKeys += nestedKeyCount - 1
					}
					flattenedKeys = append(flattenedKeys, flatKeys...)
				default:
					val, valType := parseMetadataValueType(v)
					if valType == stringType {
						if len(val) > MaxMetadataValueStringLength {
							return nil,
								0,
								fmt.Errorf(
									"metadata value exceeds maximum length of %d characters",
									MaxMetadataValueStringLength,
								)
						}
						if regexp.MustCompile(ExcludedMetadataCharactersPattern).MatchString(val) {
							excludedCharacters := strings.ReplaceAll(
								ExcludedMetadataCharactersPattern[1:len(ExcludedMetadataCharactersPattern)-1],
								"\\",
								" ",
							)
							return nil,
								0,
								fmt.Errorf(
									"metadata values can not contain the following character(s): %s",
									excludedCharacters,
								)
						}
					}
					flattenedKeys = append(flattenedKeys, model.RunMetadataIndex{FlatKey: newKey, Value: val, DataType: valType})
				}
			}
		// If the value is a primitive, add it to the flatKeys list.
		default:
			val, valType := parseMetadataValueType(value)
			if valType == stringType {
				if len(val) > MaxMetadataValueStringLength {
					return nil, 0, fmt.Errorf("metadata value exceeds maximum length of %d characters", MaxMetadataValueStringLength)
				}
				if regexp.MustCompile(ExcludedMetadataCharactersPattern).MatchString(val) {
					excludedCharacters := strings.ReplaceAll(
						ExcludedMetadataCharactersPattern[1:len(ExcludedMetadataCharactersPattern)-1],
						"\\",
						" ",
					)
					return nil,
						0,
						fmt.Errorf(
							"metadata values can not contain the following character(s): %s",
							excludedCharacters,
						)
				}
			}
			flattenedKeys = append(flattenedKeys, model.RunMetadataIndex{FlatKey: newKey, Value: val, DataType: valType})
		}
	}
	return flattenedKeys, numKeys, nil
}
