package storage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

// AddBackend adds storage backend information.
// We won't persist the "save_*_best" fields on the expconf struct.
// If the same storage backend has been persisted before then we will return the ID and
// not insert another row.
func AddBackend(
	ctx context.Context, cs *expconf.CheckpointStorageConfig,
) (model.StorageBackendID, error) {
	cs = schemas.WithDefaults(cs)
	if err := schemas.IsComplete(cs); err != nil {
		return 0, fmt.Errorf("schema is not complete: %w", err)
	}

	childTableRow, unionType := expconfToStorage(cs)

	var res model.StorageBackendID
	if err := db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().Returning("id").
			On("CONFLICT DO NOTHING").
			Model(childTableRow).
			Exec(ctx); err != nil {
			json, jsonErr := cs.MarshalJSON()
			if jsonErr != nil {
				return fmt.Errorf("adding storage backend: %w: %w", jsonErr, err)
			}
			return fmt.Errorf("adding storage backend %s: %w", string(json), err)
		}

		// ON CONFLICT DO NOTHING returns a non zero ID only when we insert a new row.
		// When we insert a new row also insert a new row in the parent table.
		if childTableRow.id() != 0 {
			unionTableRow := &storageBackendRow{}
			if _, err := tx.NewInsert().Model(unionTableRow).
				Value(unionType, "?", childTableRow.id()).
				Returning("id").
				Exec(ctx); err != nil {
				return fmt.Errorf("adding storage backend row: %w", err)
			}

			res = unionTableRow.ID
			return nil
		}

		// This case we have already inserted a parent and a child row.
		// First do a lookup for the child row then do another lookup of the parent.
		childBackendID, err := getChildBackendRows(ctx, tx, childTableRow)
		if err != nil {
			return fmt.Errorf("getting child backend row in dupe case %v: %w", childTableRow, err)
		}

		unionTableRow := &storageBackendRow{}
		if err := tx.NewSelect().Model(unionTableRow).
			Where("? = ?", bun.Safe(unionType), childBackendID).
			Scan(ctx, unionTableRow); err != nil {
			return fmt.Errorf("getting parent backend row in dupe case %v: %w", childTableRow, err)
		}

		res = unionTableRow.ID
		return nil
	}); err != nil {
		return 0, fmt.Errorf("adding storage backend: %w", err)
	}

	return res, nil
}

func getChildBackendRows(ctx context.Context, idb bun.IDB, backend storageBackend) (int, error) {
	q := idb.NewSelect().Model(backend).Column("id")
	wheres, args := getChildBackendRowWheres(backend)
	for i := 0; i < len(wheres); i++ {
		q.Where(wheres[i], args[i]...)
	}

	if err := q.Scan(ctx, backend); err != nil {
		return 0, fmt.Errorf("running storage child lookup query: %w", err)
	}
	return backend.id(), nil
}

// This is written like this so we can easily test this. Without testing the query it
// is really hard to generate test cases that will error if someone forgets to add a column here.
// Returning a *bun.SelectQuery is a good idea but harder to test, since the Where order can
// make the query generate differently.
func getChildBackendRowWheres(backend storageBackend) ([]string, [][]any) {
	var wheres []string
	var args [][]any

	addStringWhere := func(colName, v string) {
		wheres = append(wheres, "? = ?")
		args = append(args, []any{bun.Safe(colName), v})
	}
	addStringPtrWhere := func(colName string, v *string) {
		if v != nil {
			addStringWhere(colName, *v)
		} else {
			wheres = append(wheres, "? IS NULL")
			args = append(args, []any{bun.Safe(colName)})
		}
	}

	switch b := backend.(type) {
	case *storageBackendSharedFS:
		addStringWhere("host_path", b.HostPath)
		addStringPtrWhere("container_path", b.ContainerPath)
		addStringPtrWhere("checkpoint_path", b.CheckpointPath)
		addStringPtrWhere("tensorboard_path", b.TensorboardPath)
		addStringPtrWhere("storage_path", b.StoragePath)
		addStringWhere("propagation", b.Propagation)
	case *storageBackendS3:
		addStringWhere("bucket", b.Bucket)
		addStringPtrWhere("access_key", b.AccessKey)
		addStringPtrWhere("secret_key", b.SecretKey)
		addStringPtrWhere("endpoint_url", b.EndpointURL)
		addStringPtrWhere("prefix", b.Prefix)
	case *storageBackendGCS:
		addStringWhere("bucket", b.Bucket)
		addStringPtrWhere("prefix", b.Prefix)
	case *storageBackendAzure:
		addStringWhere("container", b.Container)
		addStringPtrWhere("connection_string", b.ConnectionString)
		addStringPtrWhere("account_url", b.AccountURL)
		addStringPtrWhere("credential", b.Credential)
	case *storageBackendDirectory:
		addStringWhere("container_path", b.ContainerPath)
	}

	return wheres, args
}

// Backend returns the checkpoint storage backend information.
// We won't return the "save_*_best" fields on the expconf struct.
func Backend(
	ctx context.Context, id model.StorageBackendID,
) (expconf.CheckpointStorageConfig, error) {
	var parentRow storageBackendRow
	if err := db.Bun().NewSelect().Model(&parentRow).
		Where("id = ?", id).
		Scan(ctx, &parentRow); err != nil {
		return expconf.CheckpointStorageConfig{},
			fmt.Errorf("getting storage backend ID %d: %w", id, err)
	}

	childRow := parentRow.toChildRowOnlyIDPopulated()
	if err := db.Bun().NewSelect().Model(childRow).
		Where("id = ?", childRow.id()).
		Scan(ctx, childRow); err != nil {
		return expconf.CheckpointStorageConfig{},
			fmt.Errorf("getting child of storage backend ID %d: %w", id, err)
	}

	conf := childRow.toExpconf()
	if conf == nil {
		return expconf.CheckpointStorageConfig{},
			fmt.Errorf("childRow toExpconf is nil, this is unexpected")
	}
	return *conf, nil
}

// CheckpointsGroup is a grouping of storage ID to checkpoints.
// This isn't a map[*model.StorageBackendID][]uuid.UUID since it is pretty
// easy to misuse the *model.StorageBackendID by just doing something like m[ptrs.Ptr(5)].
type CheckpointsGroup struct {
	StorageID   *model.StorageBackendID
	Checkpoints []uuid.UUID
}

// GroupCheckpoints takes a list of checkpoints and returns a mapping by storageID.
func GroupCheckpoints(
	ctx context.Context, uuids []uuid.UUID,
) ([]*CheckpointsGroup, error) {
	checkpoints, err := db.SingleDB().CheckpointByUUIDs(uuids)
	if err != nil {
		return nil, err
	}

	groups := make(map[model.StorageBackendID]*CheckpointsGroup)
	for _, c := range checkpoints {
		if c.UUID == nil {
			return nil, fmt.Errorf("got checkpoint uuids back with a nil uuid %+v", c)
		}

		id := model.StorageBackendID(0)
		if c.StorageID != nil {
			id = *c.StorageID
		}

		if _, ok := groups[id]; !ok {
			groups[id] = &CheckpointsGroup{
				StorageID: c.StorageID,
			}
		}
		groups[id].Checkpoints = append(groups[id].Checkpoints, *c.UUID)
	}

	return maps.Values(groups), nil
}
