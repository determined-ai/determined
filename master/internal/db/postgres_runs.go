package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/pkg/model"
)

// MaxRunMetadataKeyCount is the maximum number of unique metadata keys allowed on a run.
// added here to avoid circular dependency on run package.
const MaxRunMetadataKeyCount = 1000

// getRunMetadataKeys returns the unique metadata keys for a run.
func getRunMetadataKeys(ctx context.Context, rID int) ([]string, error) {
	var res []string
	err := Bun().NewSelect().Model(&res).Table("runs_metadata_index").
		Distinct().
		Column("flat_key").
		Where("run_id = ?", rID).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("querying run metadata indexes: %w", err)
	}
	return res, nil
}

// MergeRunMetadata two sets of run metadata together.
func MergeRunMetadata(current, addition map[string]interface{}) (map[string]interface{}, error) {
	for newKey, newVal := range addition {
		// we have a matched key
		if oldVal, ok := current[newKey]; ok {
			switch typedNewVal := newVal.(type) {
			// the new value is nested, so we'll eventually recursively join.
			case map[string]interface{}:
				switch typedOldVal := oldVal.(type) {
				case map[string]interface{}:
					temp, err := MergeRunMetadata(typedOldVal, typedNewVal)
					if err != nil {
						return nil, err
					}
					current[newKey] = temp
				case []interface{}:
					// if the old value is a list, but the new value is a map,
					// then we want to search the old list for maps that intersect with the current map.
					found := make(map[string]struct{})
					notFound := make(map[string]interface{})
					for nestedNewKey, nestedNewVal := range typedNewVal {
						for i, knownElem := range typedOldVal {
							switch typedKnownElem := knownElem.(type) {
							// we found a map inside the old list, check if there's an intersection
							case map[string]interface{}:
								if _, ok := typedKnownElem[nestedNewKey]; ok {
									found[nestedNewKey] = struct{}{}
									temp, err := MergeRunMetadata(typedKnownElem, map[string]interface{}{nestedNewKey: nestedNewVal})
									if err != nil {
										return nil, err
									}
									typedOldVal[i] = temp
								}
							}
						}
						if _, ok := found[nestedNewKey]; !ok {
							// a map with the same key as the new map was not found in the old list,
							// so we'll eventually append it to the list.
							notFound[nestedNewKey] = nestedNewVal
						}
					}
					typedOldVal = append(typedOldVal, notFound)
					current[newKey] = typedOldVal
				default:
					current[newKey] = append([]interface{}{oldVal}, newVal)
				}
			case []interface{}:
				// treat each element as if it was a kv pair and recursively merge the new value into the old value.
				for _, newElem := range typedNewVal {
					merged, err := MergeRunMetadata(map[string]interface{}{newKey: oldVal}, map[string]interface{}{newKey: newElem})
					if err != nil {
						return nil, err
					}
					oldVal = merged[newKey]
				}
				current[newKey] = oldVal
			default:
				switch typedOldVal := oldVal.(type) {
				case map[string]interface{}:
					current[newKey] = []interface{}{typedOldVal, newVal}
				case []interface{}:
					current[newKey] = append(typedOldVal, newVal)
				default:
					return nil, errors.Wrapf(ErrInvalidInput,
						"attempts to overwrite existing entry ('%s': %v) with new value '%v'",
						newKey,
						oldVal,
						newVal,
					)
				}
			}
		} else {
			// Add in new key-value pairs that are not in the old metadata.
			current[newKey] = newVal
		}
	}
	return current, nil
}

// updateRunMetadata is a helper function that returns the closure to update the metadata of a run.
func updateRunMetadata(
	rID int,
	flatKeySet map[string]struct{},
	rawMetadata map[string]interface{},
	flatMetadata []model.RunMetadataIndex,
	keyCount int,
	result *map[string]interface{},
) func(context.Context, bun.Tx) error {
	return func(ctx context.Context, tx bun.Tx) error {
		// use pessimistic locking to prevent concurrent updates
		type metadataRun struct {
			bun.BaseModel   `bun:"runs"`
			ProjectID       int
			Metadata        map[string]interface{}
			NumMetadataKeys int
		}
		run := &metadataRun{}
		err := tx.NewRaw(`
			SELECT 
				project_id, 
				metadata, 
				length(metadata::TEXT) - length(replace(metadata::text, ':', ''))
				/ length(':') AS num_metadata_keys
			FROM 
				runs 
			WHERE 
				id = ? 
			FOR UPDATE`, rID).Scan(ctx, run)
		if err != nil {
			return fmt.Errorf("querying run metadata: %w", err)
		}
		if run.NumMetadataKeys+keyCount > MaxRunMetadataKeyCount {
			return status.Errorf(
				codes.InvalidArgument,
				"request exceeds run metadata key count limit %d/%d on run(%d)",
				(run.NumMetadataKeys + keyCount),
				MaxRunMetadataKeyCount,
				rID,
			)
		}
		if run.Metadata == nil {
			run.Metadata = make(map[string]interface{})
		}

		// check for duplicate keys
		currentKeys, err := getRunMetadataKeys(ctx, rID)
		if err != nil {
			return fmt.Errorf("getting run metadata keys: %w", err)
		}
		duplicateKeys := []string{}
		for _, key := range currentKeys {
			if _, ok := flatKeySet[key]; ok {
				duplicateKeys = append(duplicateKeys, key)
			}
		}
		if len(duplicateKeys) > 0 {
			return status.Errorf(
				codes.InvalidArgument,
				"following metadata key(s) already exist on run(%d): %q",
				rID,
				duplicateKeys,
			)
		}

		// merge the new metadata with the existing metadata
		temp, err := MergeRunMetadata(run.Metadata, rawMetadata)
		if err != nil {
			if errors.Is(err, ErrInvalidInput) {
				return status.Errorf(codes.InvalidArgument, err.Error())
			}
			return errors.Wrapf(err, "merging run(%d) metadata: %v", rID, err)
		}
		*result = temp
		_, err = tx.NewUpdate().Table("runs").Set("metadata = ?", *result).Where("id = ?", rID).Exec(ctx)
		if err != nil {
			return fmt.Errorf("updating run metadata on run(%d): %w", rID, err)
		}

		// hydrate the flat metadata with relevant ids.
		for i := range flatMetadata {
			flatMetadata[i].RunID = rID
			flatMetadata[i].ProjectID = run.ProjectID
		}

		_, err = tx.NewInsert().Model(&flatMetadata).Exec(ctx)
		if err != nil {
			return fmt.Errorf("inserting run metadata indexes for run(%d): %w", rID, err)
		}
		return nil
	}
}

// UpdateRunMetadata updates the metadata of a run, including the metadata indexes.
func UpdateRunMetadata(
	ctx context.Context,
	rID int,
	rawMetadata map[string]interface{},
	flatKeySet map[string]struct{},
	keyCount int,
	flatMetadata []model.RunMetadataIndex,
) (result map[string]interface{}, err error) {
	err = Bun().RunInTx(
		ctx,
		&sql.TxOptions{Isolation: sql.LevelReadCommitted},
		updateRunMetadata(rID, flatKeySet, rawMetadata, flatMetadata, keyCount, &result),
	)

	if err != nil {
		return nil, err
	}
	return result, nil
}
