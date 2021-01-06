package expconf

import (
	"fmt"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

// ParseAnyExperimentConfigJSON will return the latest-available experiment config version, even if
// it is passed the oldest-supported version to unmarshal.  It returns a user-facing error if there
// is an issue in the process.
func ParseAnyExperimentConfigJSON(byts []byte) (ExperimentConfig, error) {
	var v0 ExperimentConfigV0

	var versioned struct {
		Version int `json:"version"`
	}

	// Detect version
	err := json.Unmarshal(byts, &versioned)
	if err != nil {
		return v0, errors.Wrap(err, "unable to unmarshal json-formatted experiment config")
	}
	version := versioned.Version

	// versioned parsing
	switch version {
	case 0:
		err = json.Unmarshal(byts, &v0)
		if err != nil {
			return v0, errors.Wrap(err, "unable to unmarshal experiment config as V0")
		}

	default:
		return v0, errors.New(fmt.Sprintf("invalid version: %d", version))
	}

	return v0, nil
}

// ParseAnyExperimentConfigYAML just wraps ParseAnyExperimentConfigJSON
func ParseAnyExperimentConfigYAML(byts []byte) (ExperimentConfig, error) {
	byts, err := schemas.JsonFromYaml([]byte(byts))
	if err != nil {
		return ExperimentConfig{}, errors.Wrap(err, "unable to convert yaml to json")
	}
	return ParseAnyExperimentConfigJSON(byts)
}
