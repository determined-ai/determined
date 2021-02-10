package expconf

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

// ParseAnyExperimentConfigJSON will return the latest-available experiment config version, even if
// it is passed the oldest-supported version to unmarshal.  It returns a user-facing error if there
// is an issue in the process.
//
// ParseAnyExperimentConfigJSON will ensure that the bytes are sane for the appropriate version of
// the experiment config before unmarshaling, but it will not check if the config is complete,
// since cluster-level settings and defaults will not have been set yet.
func ParseAnyExperimentConfigJSON(byts []byte) (ExperimentConfig, error) {
	var out ExperimentConfigV0
	// TODO: uncomment lines in this function to support automatic version detection + shimming.
	// var v0 ExperimentConfigV0
	// var out ExperimentConfigV1

	var versioned struct {
		Version int `json:"version"`
	}

	// Detect version
	err := json.Unmarshal(byts, &versioned)
	if err != nil {
		return out, errors.Wrap(err, "unable to unmarshal json-formatted experiment config")
	}
	version := versioned.Version

	// versioned parsing
	switch version {
	case 0:
		err = schemas.SaneBytes(&out, byts)
		// err = schemas.SaneBytes(&v0, byts)
		if err != nil {
			return out, errors.Wrap(err, "version 0 experiment config is invalid")
		}
		err = json.Unmarshal(byts, &out)
		if err != nil {
			return out, errors.Wrap(err, "unable to unmarshal experiment config as version 0")
		}

	// case 1:
	// 	err = schemas.SaneBytes(&out, byts)
	// 	if err != nil {
	// 		return out, errors.Wrap(err, "version 1 experiment config is invalid")
	// 	}
	// 	err = json.Unmarshal(byts, &out)
	// 	if err != nil {
	// 		return out, errors.Wrap(err, "unable to unmarshal experiment config as version 1")
	// 	}

	default:
		return out, errors.New(fmt.Sprintf("invalid version: %d", version))
	}

	// Call shim on each old versions, walking our way to the latest version.
	// if version < 1 {
	// 	err := v0.shim(&out)
	// 	if err != nil {
	// 		return out, errors.Wrap(err, "unable to shim v0 config to v1 config")
	// 	}
	// }

	return out, nil
}

// ParseAnyExperimentConfigYAML just wraps ParseAnyExperimentConfigJSON
func ParseAnyExperimentConfigYAML(byts []byte) (ExperimentConfig, error) {
	byts, err := schemas.JSONFromYaml(byts)
	if err != nil {
		return ExperimentConfig{}, errors.Wrap(err, "unable to convert yaml to json")
	}
	return ParseAnyExperimentConfigJSON(byts)
}
