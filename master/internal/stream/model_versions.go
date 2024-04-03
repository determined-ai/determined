package stream

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

const (
	// ModelVersionsDeleteKey specifies the key for delete model_versions.
	ModelVersionsDeleteKey = "modelversions_deleted"
	// ModelVersionsUpsertKey specifies the key for upsert model_versions.
	ModelVersionsUpsertKey = "modelversion"
	// model_versionChannel specifies the channel to listen to model_version events.
	modelVersionChannel = "stream_model_version_chan"
)

// ModelVersionMsg is a stream.Msg.
//
// determined:stream-gen source=server delete_msg=ModelVersionsDeleted
type ModelVersionMsg struct {
	bun.BaseModel `bun:"table:model_versions"`

	// immutable attributes
	ID int `bun:"id,pk" json:"id"`

	// mutable attributes
	Name            string    `bun:"name" json:"name"`
	Version         int       `bun:"version" json:"version"`
	CheckpointUUID  string    `bun:"checkpoint_uuid" json:"checkpoint_uuid"`
	CreationTime    time.Time `bun:"creation_time" json:"creation_time"`
	LastUpdatedTime time.Time `bun:"last_updated_time" json:"last_updated_time"`
	Metadata        JSONB     `bun:"metadata,type:jsonb" json:"metadata"`
	ModelID         int       `bun:"model_id" json:"model_id"`
	UserID          int       `bun:"user_id" json:"user_id"`
	Comment         string    `bun:"comment" json:"comment"`
	Labels          []string  `bun:"labels,array" json:"labels"`
	Notes           string    `bun:"notes" json:"notes"`
	WorkspaceID     string    `json:"workspace_id"`
	// metadata
	Seq int64 `bun:"seq" json:"seq"`
}

// SeqNum gets the SeqNum from a ModelVersionMsg.
func (pm *ModelVersionMsg) SeqNum() int64 {
	return pm.Seq
}

// UpsertMsg creates a ModelVersion stream upsert message.
func (pm *ModelVersionMsg) UpsertMsg() stream.UpsertMsg {
	return stream.UpsertMsg{
		JSONKey: ModelVersionsUpsertKey,
		Msg:     pm,
	}
}

// DeleteMsg creates a ModelVersion stream delete message.
func (pm *ModelVersionMsg) DeleteMsg() stream.DeleteMsg {
	deleted := strconv.FormatInt(int64(pm.ID), 10)
	return stream.DeleteMsg{
		Key:     ModelVersionsDeleteKey,
		Deleted: deleted,
	}
}

// ModelVersionSubscriptionSpec is what a user submits to define a model_version subscription.
//
// determined:stream-gen source=client
type ModelVersionSubscriptionSpec struct {
	ModelIDs        []int `json:"model_ids"`
	ModelVersionIDs []int `json:"model_version_ids"`
	UserIDs         []int `json:"user_ids"`
	Since           int64 `json:"since"`
}

func modelVersionPermFilterQuery(q *bun.SelectQuery, accessScopes []model.AccessScopeID,
) *bun.SelectQuery {
	return q.Join("JOIN models ON models.id = model_id").Where("workspace_id in (?)", bun.In(accessScopes))
}

// createFilteredModelVersionIDQuery creates a select query that
// pulls all relevant model_version ids based on permission scope and
// subscription spec filters.
func createFilteredModelVersionIDQuery(
	globalAccess bool,
	accessScopes []model.AccessScopeID,
	spec ModelVersionSubscriptionSpec,
) *bun.SelectQuery {
	q := db.Bun().NewSelect().
		TableExpr("model_versions m").
		Column("m.id").
		OrderExpr("m.id ASC")

	// add permission scope filter in event of non-global access
	if !globalAccess {
		q = modelVersionPermFilterQuery(q, accessScopes)
	}

	q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		if len(spec.ModelVersionIDs) > 0 {
			q.WhereOr("m.id in (?)", bun.In(spec.ModelVersionIDs))
		}
		if len(spec.ModelIDs) > 0 {
			q.WhereOr("model_id in (?)", bun.In(spec.ModelIDs))
		}
		if len(spec.UserIDs) > 0 {
			q.WhereOr("user_id in (?)", bun.In(spec.UserIDs))
		}
		return q
	})
	return q
}

// ModelVersionCollectStartupMsgs collects ModelVersionMsg's that were missed prior to startup.
// nolint: dupl
func ModelVersionCollectStartupMsgs(
	ctx context.Context,
	user model.User,
	known string,
	spec ModelVersionSubscriptionSpec,
) (
	[]stream.MarshallableMsg, error,
) {
	var out []stream.MarshallableMsg

	if len(spec.ModelVersionIDs) == 0 && len(spec.ModelIDs) == 0 && len(spec.UserIDs) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, stream.DeleteMsg{
			Key:     ModelVersionsDeleteKey,
			Deleted: known,
		})
		return out, nil
	}
	// step 0: get user's permitted access scopes
	accessMap, err := AuthZProvider.Get().GetModelVersionStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}
	globalAccess, accessScopes := getStreamableScopes(accessMap)

	// step 1: calculate all ids matching this subscription
	createQuery := func() *bun.SelectQuery {
		return createFilteredModelVersionIDQuery(
			globalAccess,
			accessScopes,
			spec,
		)
	}
	missing, appeared, err := processQuery(ctx, createQuery, spec.Since, known, "m")
	if err != nil {
		return nil, err
	}

	// step 2: hydrate appeared IDs into full ModelVersionMsgs
	var mvMsgs []*ModelVersionMsg
	if len(appeared) > 0 {
		query := db.Bun().NewSelect().Model(&mvMsgs).
			ExcludeColumn("workspace_id").
			Where("model_version_msg.id in (?)", bun.In(appeared))
		if !globalAccess {
			query = modelVersionPermFilterQuery(query, accessScopes)
		}
		err := query.Scan(ctx, &mvMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 3: emit deletions and updates to the client
	out = append(out, stream.DeleteMsg{
		Key:     ModelVersionsDeleteKey,
		Deleted: missing,
	})
	for _, msg := range mvMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// ModelVersionMakeFilter creates a ModelVersionMsg filter based on the given ModelVersionSubscriptionSpec.
func ModelVersionMakeFilter(spec *ModelVersionSubscriptionSpec) (func(*ModelVersionMsg) bool, error) {
	// should this filter even run?
	if len(spec.ModelVersionIDs) == 0 && len(spec.ModelIDs) == 0 && len(spec.UserIDs) == 0 {
		return nil, errors.Errorf("invalid subscription spec arguments: %v %v %v",
			spec.ModelVersionIDs, spec.ModelIDs, spec.UserIDs)
	}

	// create sets based on subscription spec
	modelIDs := make(map[int]struct{})
	for _, id := range spec.ModelIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid workspace id: %d", id)
		}
		modelIDs[id] = struct{}{}
	}
	modelVersionIDs := make(map[int]struct{})
	for _, id := range spec.ModelVersionIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid model_version id: %d", id)
		}
		modelVersionIDs[id] = struct{}{}
	}
	userIDs := make(map[int]struct{})
	for _, id := range spec.UserIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid user id: %d", id)
		}
		userIDs[id] = struct{}{}
	}

	// return a closure around our copied maps
	return func(msg *ModelVersionMsg) bool {
		// subscribed to model_version by this model_version_id?
		if _, ok := modelVersionIDs[msg.ID]; ok {
			return true
		}
		// subscribed to this model_version by model_id?
		if _, ok := modelIDs[msg.ModelID]; ok {
			return true
		}
		// subscribed to this model_version by user_id?
		if _, ok := userIDs[msg.UserID]; ok {
			return true
		}
		return false
	}, nil
}

// ModelVersionMakePermissionFilter returns a function that checks if a ModelVersionMsg
// is in scope of the user permissions.
func ModelVersionMakePermissionFilter(ctx context.Context, user model.User) (func(*ModelVersionMsg) bool, error) {
	accessScopeSet, err := AuthZProvider.Get().GetModelVersionStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}

	switch {
	case accessScopeSet[model.GlobalAccessScopeID]:
		// user has global access for viewing model_versions
		return func(msg *ModelVersionMsg) bool {
			return true
		}, nil
	default:
		return func(msg *ModelVersionMsg) bool {
			workspaceID, _ := strconv.Atoi(msg.WorkspaceID)
			return accessScopeSet[model.AccessScopeID(workspaceID)]
		}, nil
	}
}
