package expconf

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/ptrs"
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
	CheckpointStorage CheckpointStorageConfig
	BindMounts        BindMountsConfig
	Environment       EnvironmentConfig
	Hyperparameters   Hyperparameters
	Searcher          LegacySearcher
}

// LegacySearcher represents a subset of the SearcherConfig which can be expected to be available
// for all experiments, new and old.
type LegacySearcher struct {
	// Name might contain an EOL searcher name.
	Name            string
	Metric          string
	SmallerIsBetter bool
}

func getCheckpointStorage(raw map[string]interface{}) (CheckpointStorageConfig, error) {
	var cs CheckpointStorageConfig

	csOnly := raw["checkpoint_storage"]

	// Special case for hdfs.
	switch m := csOnly.(type) {
	case map[string]any:
		if t, ok := m["type"].(string); ok && t == "hdfs" {
			var saveExpBest, saveTrialBest, saveTrialLatest *int
			var i float64
			if i, ok = m["save_experiment_best"].(float64); ok {
				saveExpBest = ptrs.Ptr(int(i))
			}
			if i, ok = m["save_trial_best"].(float64); ok {
				saveTrialBest = ptrs.Ptr(int(i))
			}
			if i, ok = m["save_trial_latest"].(float64); ok {
				saveTrialLatest = ptrs.Ptr(int(i))
			}

			// nolint: exhaustivestruct
			dummyHDFSSharedFS := schemas.WithDefaults(CheckpointStorageConfig{
				RawSharedFSConfig: &SharedFSConfig{
					RawHostPath: ptrs.Ptr("/legacy-hdfs-checkpoint-path"),
				},
				RawSaveExperimentBest: saveExpBest,
				RawSaveTrialBest:      saveTrialBest,
				RawSaveTrialLatest:    saveTrialLatest,
			})

			if err := schemas.IsComplete(dummyHDFSSharedFS); err != nil {
				return cs, fmt.Errorf("shared fs shim for hdfs is incomplete: %w", err)
			}
			return dummyHDFSSharedFS, nil
		}
	}

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
	cs = schemas.WithDefaults(cs)

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
	bm = schemas.WithDefaults(bm)
	if err = schemas.IsComplete(bm); err != nil {
		return bm, errors.Wrap(err, "legacy bind mounts is incomplete")
	}
	return bm, nil
}

func getEnvironment(raw map[string]interface{}) (EnvironmentConfig, error) {
	var env EnvironmentConfig

	envOnly := raw["environment"]
	if envOnly != nil {
		envByts, err := json.Marshal(envOnly)
		if err != nil {
			return env, errors.Wrap(err, "unable to remarshal environment as json")
		}
		if err = schemas.SaneBytes(&env, envByts); err != nil {
			return env, errors.Wrap(err, "legacy environment does not pass sanity checks")
		}
		if err = json.Unmarshal(envByts, &env); err != nil {
			return env, errors.Wrap(err, "unable to unmarshal environment bytes")
		}
	}

	env = schemas.WithDefaults(env)

	if err := schemas.IsComplete(env); err != nil {
		return env, errors.Wrap(err, "legacy environment is incomplete")
	}

	return env, nil
}

func getHyperparameters(raw map[string]interface{}) (Hyperparameters, error) {
	h := Hyperparameters{}

	hpOnly := raw["hyperparameters"]
	if hpOnly == nil {
		// Empty hyperparameters.
		return h, nil
	}

	hpBytes, err := json.Marshal(hpOnly)
	if err != nil {
		return h, errors.Wrap(err, "unable to remarshal hyperparameters as json")
	}
	if err = schemas.SaneBytes(&h, hpBytes); err != nil {
		return h, errors.Wrap(err, "legacy hyperparameters do not pass sanity checks")
	}
	if err = json.Unmarshal(hpBytes, &h); err != nil {
		return h, errors.Wrap(err, "unable to unmarshal hyperparameter bytes")
	}
	h = schemas.WithDefaults(h)
	if err = schemas.IsComplete(h); err != nil {
		return h, errors.Wrap(err, "legacy hyperparameters are incomplete")
	}
	return h, nil
}

func getLegacySearcher(raw map[string]interface{}) (LegacySearcher, error) {
	searcher := raw["searcher"]
	if searcher == nil {
		return LegacySearcher{}, errors.New("searcher field missing")
	}

	tsearcher, ok := searcher.(map[string]interface{})
	if !ok {
		return LegacySearcher{}, errors.New("searcher field is not a map")
	}

	name, ok := tsearcher["name"]
	if !ok {
		return LegacySearcher{}, errors.New("searcher.name missing")
	}

	tname, ok := name.(string)
	if !ok {
		return LegacySearcher{}, errors.New("searcher.name is not a string")
	}

	metric, ok := tsearcher["metric"]
	if !ok {
		return LegacySearcher{}, errors.New("searcher.metric missing")
	}

	tmetric, ok := metric.(string)
	if !ok {
		return LegacySearcher{}, errors.New("searcher.metric is not a string")
	}

	// smallerIsBetter has always had a default, and always the same one
	tsmallerIsBetter := true
	if smallerIsBetter, ok := tsearcher["smaller_is_better"]; ok && smallerIsBetter != nil {
		tsmallerIsBetter, ok = smallerIsBetter.(bool)
		if !ok {
			return LegacySearcher{}, errors.New("searcher.smaller_is_better is not a boolean")
		}
	}

	return LegacySearcher{
		Name:            tname,
		Metric:          tmetric,
		SmallerIsBetter: tsmallerIsBetter,
	}, nil
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
	var out LegacyConfig

	raw := map[string]interface{}{}
	if err := json.Unmarshal(byts, &raw); err != nil {
		return out, errors.Wrap(err, "unable to unmarshal bytes as json at all")
	}

	cs, err := getCheckpointStorage(raw)
	if err != nil {
		return out, err
	}
	out.CheckpointStorage = cs

	bm, err := getBindMounts(raw)
	if err != nil {
		return out, err
	}
	out.BindMounts = bm

	env, err := getEnvironment(raw)
	if err != nil {
		return out, err
	}
	out.Environment = env

	hp, err := getHyperparameters(raw)
	if err != nil {
		return out, err
	}
	out.Hyperparameters = hp

	searcher, err := getLegacySearcher(raw)
	if err != nil {
		return out, err
	}
	out.Searcher = searcher

	return out, nil
}

// Scan implements the db.Scanner interface.
func (l *LegacyConfig) Scan(src interface{}) error {
	byts, ok := src.([]byte)
	if !ok {
		return errors.Errorf("unable to convert to []byte: %v", src)
	}
	config, err := ParseLegacyConfigJSON(byts)
	if err != nil {
		return err
	}
	*l = config
	return nil
}
