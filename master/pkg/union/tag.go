package union

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

const unionTag = "union"

// parseUnionStructTag parses the "union" struct tag. The format of the struct tag is "key,value"
// where key is the key is the common union type name for all the union type values and value is
// the name of the fields union type.
func parseUnionStructTag(tagValue string) (string, string, error) {
	switch parsed := strings.Split(tagValue, ","); {
	case len(parsed) == 2:
		return parsed[0], parsed[1], nil
	default:
		return "", "", errors.Errorf("unexpected union tag format: %s", unionTag)
	}
}

// getTagValue returns the name of the union type (keyed by the tag field) that is defined in the
// data bytes. If no key is defined, the second result returns false. If input data is not a JSON
// object or the tag value is not a string, an error is returned.
func getTagValue(data []byte, tag string) (string, bool, error) {
	// Parse the raw JSON blob into a map.
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", false, err
	}

	// Look for the key tag in the map.
	tagValue, ok := parsed[tag]
	if !ok {
		return "", false, nil
	}

	// Ensure that the tag value is a string.
	typed, ok := tagValue.(string)
	if !ok {
		return "", false, errors.Errorf("%s must be a string: got %T", tag, typed)
	}
	return typed, true, nil
}
