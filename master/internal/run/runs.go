package run

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
)

const (
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
	// MaxKeyCount is the maximum number of metadata keys allowed for a run.
	MaxKeyCount = 1000
)

// FlattenMetadata flattens a nested map of run metadata into a list of RunMetadataIndex entries.
func FlattenMetadata(
	data map[string]any,
) (flatMetadata []model.RunMetadataIndex, err error) {
	if len(data) == 0 {
		return nil, nil
	}
	flatMetadata, err = flattenMetadata(data)
	if err != nil {
		return nil, fmt.Errorf("error flattening metadata: %w", err)
	}
	return flatMetadata, nil
}

// parseTimestamp converts a string to a timestamp and returns the timestamp.
func parseTimestamp(value string) (string, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02",
		"2006-01",
	}
	for _, format := range formats {
		if timestamp, err := time.Parse(format, value); err == nil {
			return timestamp.UTC().Format(time.RFC3339Nano), nil
		}
	}
	return "", fmt.Errorf("unable to parse timestamp")
}

func flattenMetadata(data map[string]any) (flatMetadata []model.RunMetadataIndex, err error) {
	type metadataEntry struct {
		prefix string
		key    string
		value  any
		depth  int
		array  bool
	}
	stack := []metadataEntry{{prefix: "", key: "", value: data, depth: 0, array: false}}

	// terminate early if we exceed the key count.
	for len(stack) > 0 && len(flatMetadata) <= MaxKeyCount {
		// pop an entry from the stack.
		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// validate the entry
		switch {
		case entry.depth > MaxMetadataDepth:
			return nil, fmt.Errorf("metadata exceeds maximum nesting depth of %d", MaxMetadataDepth)
		case len(entry.key) > MaxMetadataKeyLength:
			return nil, fmt.Errorf("metadata key exceeds maximum length of %d characters", MaxMetadataKeyLength)
		case regexp.MustCompile(ExcludedMetadataCharactersPattern).MatchString(entry.key):
			excludedCharacters := strings.ReplaceAll(
				ExcludedMetadataCharactersPattern[1:len(ExcludedMetadataCharactersPattern)-1],
				"\\",
				" ",
			)
			return nil, fmt.Errorf(
				"metadata keys can not contain the following character(s): %s",
				excludedCharacters,
			)
		}

		switch typedVal := entry.value.(type) {
		// if the value is a map, push each key-value pair onto the stack.
		case map[string]any:
			if len(typedVal) == 0 {
				// if the map is empty, treat it as a leaf node with a nil value.
				newIndex := model.RunMetadataIndex{
					FlatKey:        entry.prefix + entry.key,
					IsArrayElement: false,
				}
				flatMetadata = append(flatMetadata, newIndex)
				continue
			}
			for key, value := range typedVal {
				newPrefix := entry.prefix + entry.key + "."
				if newPrefix == "." {
					newPrefix = ""
				}
				// push the key-value pair onto the stack
				stack = append(stack, metadataEntry{
					prefix: newPrefix,
					key:    key,
					value:  value,
					depth:  entry.depth + 1,
					array:  entry.array,
				})
			}
		// if the value is a slice, push each element onto the stack.
		case []any:
			if len(typedVal) == 0 {
				// if the slice is empty, treat it as a leaf node with a nil value.
				newIndex := model.RunMetadataIndex{
					FlatKey:        entry.prefix + entry.key,
					IsArrayElement: true,
				}
				flatMetadata = append(flatMetadata, newIndex)
				continue
			}
			if len(typedVal) > MaxMetadataArrayLength {
				return nil, fmt.Errorf(
					"metadata array exceeds maximum length of %d/%d elements",
					len(typedVal),
					MaxMetadataArrayLength,
				)
			}
			for _, value := range typedVal {
				stack = append(stack, metadataEntry{
					prefix: entry.prefix,
					key:    entry.key,
					value:  value,
					depth:  entry.depth + 1,
					array:  true,
				})
			}
		// if the value is a primitive or an unknown type, treat it as a leaf node.
		default:
			newIndex := model.RunMetadataIndex{
				FlatKey:        entry.prefix + entry.key,
				IsArrayElement: entry.array,
			}
			// parse the value and set the appropriate field.
			switch v := entry.value.(type) {
			case int:
				newIndex.IntegerValue = &v
			case float64:
				if v == float64(int(v)) {
					newIndex.IntegerValue = ptrs.Ptr(int(v))
				} else {
					newIndex.FloatValue = &v
				}
			case bool:
				newIndex.BooleanValue = &v
			case string:
				if timestampVal, err := parseTimestamp(v); err == nil {
					newIndex.TimestampValue = &timestampVal
				} else {
					if len(v) > MaxMetadataValueStringLength {
						return nil, fmt.Errorf(
							"metadata value exceeds maximum length of %d characters",
							MaxMetadataValueStringLength,
						)
					}
					newIndex.StringValue = &v
				}
			}
			flatMetadata = append(flatMetadata, newIndex)
		}
	}

	// check if we exceeded the key count
	if len(flatMetadata) > MaxKeyCount {
		return nil, fmt.Errorf("request exceeds run metadata key count limit ofs %d", MaxKeyCount)
	}
	return flatMetadata, nil
}
