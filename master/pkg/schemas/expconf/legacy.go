package expconf

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

// LegacyConfig represents a config that was once valid but might no longer be shimmable to the
// current version of ExperimentConfig.  Fortunately, the parts of the system that need to read
// configs that are this old are either 1) only concerned with the raw config for rendering it to
// the user, or 2) only need very specific parts of the old config, so to reduce our own burden we
// only expose very specific data from it.
//
// LegacyConfig can be used to deal with configs that contain some EOL components, like searchers
// that have been removed.
type LegacyConfig struct {
	checkpointStorage CheckpointStorageConfig
	bindMounts        BindMountsConfig
	envvars           EnvironmentVariablesMap
	podSpec           *PodSpec
}

// CheckpointStorage returns a current CheckpointStorage from a LegacyConfig.
func (h LegacyConfig) CheckpointStorage() CheckpointStorageConfig {
	return h.checkpointStorage
}

// BindMounts returns a current BindMountsConfig from a LegacyConfig.
func (h LegacyConfig) BindMounts() BindMountsConfig {
	return h.bindMounts
}

// EnvironmentVariables returns a current EnvironmentVariables from a LegacyConfig.
func (h LegacyConfig) EnvironmentVariables() EnvironmentVariablesMap {
	return h.envvars
}

// PodSpec returns a current k8s PodSpec from a LegacyConfig.
func (h LegacyConfig) PodSpec() *PodSpec {
	return h.podSpec
}

func getCheckpointStorage(raw map[string]interface{}) (CheckpointStorageConfig, error) {
	cs := CheckpointStorageConfig{}

	csOnly := raw["checkpoint_storage"]

	csByts, err := json.Marshal(csOnly)
	if err != nil {
		return cs, errors.Wrap(err, "unable to remarshal config bytes as json")
	}

	// Read the checkpoint storage.
	if err = schemas.SaneBytes(&cs, csByts); err != nil {
		return cs, errors.Wrap(err, "legacy checkpoint storage does not pass sanity checks")
	}
	if err = json.Unmarshal(csByts, &cs); err != nil {
		return cs, errors.Wrap(err, "unable to unmarshal checkpoint storage bytes")
	}

	// Fill defaults (should be a no-op).
	cs = schemas.WithDefaults(cs).(CheckpointStorageConfig)

	// Validate fully before passing anything out.
	if err = schemas.IsComplete(cs); err != nil {
		return cs, errors.Wrap(err, "legacy checkpoint storage is incomplete")
	}

	return cs, nil
}

func getBindMounts(raw map[string]interface{}) (BindMountsConfig, error) {
	bm := BindMountsConfig{}

	bmOnly := raw["bind_mounts"]
	if bmOnly == nil {
		// Empty bind_mounts.
		return bm, nil
	}

	bmByts, err := json.Marshal(bmOnly)
	if err != nil {
		return bm, errors.Wrap(err, "unable to remarshal bind mounts as json")
	}
	if err = schemas.SaneBytes(&bm, bmByts); err != nil {
		return bm, errors.Wrap(err, "legacy bind mounts does not pass sanity checks")
	}
	if err = json.Unmarshal(bmByts, &bm); err != nil {
		return bm, errors.Wrap(err, "unable to unmarshal bind mounts bytes")
	}
	bm = schemas.WithDefaults(bm).(BindMountsConfig)
	if err = schemas.IsComplete(bm); err != nil {
		return bm, errors.Wrap(err, "legacy bind mounts is incomplete")
	}
	return bm, nil
}

func getEnvironmentVariables(raw map[string]interface{}) (EnvironmentVariablesMap, error) {
	ev := EnvironmentVariablesMap{}

	envOnly := raw["environment"]
	if envOnly == nil {
		// Empty environment.
		return ev, nil
	}

	evOnly := envOnly.(map[string]interface{})["environment_variables"]
	if evOnly == nil {
		// Empty environment.environment_variables.
		return ev, nil
	}

	evByts, err := json.Marshal(evOnly)
	if err != nil {
		return ev, errors.Wrap(err, "unable to remarshal environment variables as json")
	}

	// The EnvironemntVariablesMap object doesn't point to quite the right schema to validate
	// against (it can't handle plain lists), so we manually specify the more general schema.
	validator := schemas.GetSanityValidator(
		"http://determined.ai/schemas/expconf/v0/environment-variables.json",
	)
	if err = validator.Validate(bytes.NewReader(evByts)); err != nil {
		err = errors.New(schemas.JoinErrors(schemas.GetRenderedErrors(err, evByts), "\n"))
		return ev, errors.Wrap(err, "legacy environment variables does not pass sanity checks")
	}

	if err = json.Unmarshal(evByts, &ev); err != nil {
		return ev, errors.Wrap(err, "unable to unmarshal environment variables bytes")
	}
	ev = schemas.WithDefaults(ev).(EnvironmentVariablesMap)

	// The unmarshaling will convert plain lists into a map of lists, so the normal json-schema
	// API patterns (schemas.IsComplete) will now work.
	if err = schemas.IsComplete(ev); err != nil {
		return ev, errors.Wrap(err, "legacy environment variables is incomplete")
	}
	return ev, nil
}

func getPodSpec(raw map[string]interface{}) (*PodSpec, error) {
	ps := &PodSpec{}

	envOnly := raw["environment"]
	if envOnly == nil {
		return nil, nil
	}

	psOnly := envOnly.(map[string]interface{})["pod_spec"]
	if psOnly == nil {
		return nil, nil
	}

	rawBytes, err := json.Marshal(psOnly)
	if err != nil {
		return nil, errors.Wrap(err, "unable to remarshal pod spec as json")
	}

	if err = json.Unmarshal(rawBytes, ps); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal pod spec bytes")
	}

	return ps, nil
}

// ParseLegacyConfigJSON parses bytes that represent an experiment config that was once valid
// but might no longer be shimmable to the current version of ExperimentConfig.
func ParseLegacyConfigJSON(byts []byte) (LegacyConfig, error) {
	// Known difficulties (as of 0.15.5) that we are implicitly avoiding here:
	// - Pre-remove-steps configs cannot be parsed as ExperimentConfig because several
	//   fields changed, including but not limited to batches_per_step, searcher.max_length.
	//   We are implicitly avoiding this by simply stripping all fields except checkpoint_storage.
	// - There was a removal of adaptive and adaptive_simple.  As of 0.15.5, some of that is handled
	//   by reading select parts of the searcher config in SQL queries.  Eventually that may get
	//   handled here, in which case we need to be aware of that removal.
	out := LegacyConfig{}

	raw := map[string]interface{}{}
	if err := json.Unmarshal(byts, &raw); err != nil {
		return out, errors.Wrap(err, "unable to unmarshal bytes as json at all")
	}

	cs, err := getCheckpointStorage(raw)
	if err != nil {
		return out, err
	}
	out.checkpointStorage = cs

	bm, err := getBindMounts(raw)
	if err != nil {
		return out, err
	}
	out.bindMounts = bm

	ev, err := getEnvironmentVariables(raw)
	if err != nil {
		return out, err
	}
	out.envvars = ev

	ps, err := getPodSpec(raw)
	if err != nil {
		return out, err
	}
	out.podSpec = ps

	return out, nil
}
