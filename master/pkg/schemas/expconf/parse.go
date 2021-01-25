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
	var v1 ExperimentConfigV1

	var versioned struct {
		Version int `json:"version"`
	}

	// Detect version
	err := json.Unmarshal(byts, &versioned)
	if err != nil {
		return v1, errors.Wrap(err, "unable to unmarshal json-formatted experiment config")
	}
	version := versioned.Version

	// versioned parsing
	switch version {
	case 0:
		err = json.Unmarshal(byts, &v0)
		if err != nil {
			return v1, errors.Wrap(err, "unable to unmarshal experiment config as V0")
		}

	case 1:
		err = json.Unmarshal(byts, &v1)
		if err != nil {
			return v1, errors.Wrap(err, "unable to unmarshal experiment config as V1")
		}

	default:
		return v1, errors.New(fmt.Sprintf("invalid version: %d", version))
	}

	// Call shim on each old versions, walking our way to the latest version.
	if version == 0 {
		err := v0.shim(&v1)
		if err != nil {
			return v1, errors.Wrap(err, "unable to shim v0 config to v1 config")
		}
		version++
	}

	return v1, nil
}

// ParseAnyExperimentConfigYAML just wraps ParseAnyExperimentConfigJSON
func ParseAnyExperimentConfigYAML(byts []byte) (ExperimentConfig, error) {
	byts, err := schemas.JsonFromYaml([]byte(byts))
	if err != nil {
		return ExperimentConfig{}, errors.Wrap(err, "unable to convert yaml to json")
	}
	return ParseAnyExperimentConfigJSON(byts)
}
