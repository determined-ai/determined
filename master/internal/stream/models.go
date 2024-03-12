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
	// ModelsDeleteKey specifies the key for delete models.
	ModelsDeleteKey = "models_deleted"
	// ModelsUpsertKey specifies the key for upsert models.
	ModelsUpsertKey = "model"
	// modelChannel specifies the channel to listen to model events.
	modelChannel = "stream_model_chan"
)

// ModelMsg is a stream.Msg.
//
// determined:stream-gen source=server delete_msg=ModelsDeleted
type ModelMsg struct {
	bun.BaseModel `bun:"table:models"`

	// immutable attributes
	ID int `bun:"id,pk" json:"id"`

	// mutable attributes
	Name            string    `bun:"name" json:"name"`
	Description     string    `bun:"description" json:"description"`
	Archived        bool      `bun:"archived" json:"archived"`
	CreationTime    time.Time `bun:"creation_time" json:"creation_time"`
	Notes           JSONB     `bun:"notes" json:"notes"`
	WorkspaceID     int       `bun:"workspace_id" json:"workspace_id"`
	UserID          int       `bun:"user_id" json:"user_id"`
	LastUpdatedTime time.Time `bun:"last_updated_time" json:"last_updated_time"`
	Metadata        JSONB     `bun:"metadata" json:"metadata"`
	Labels          JSONB     `bun:"labels" json:"labels"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`
}

// SeqNum gets the SeqNum from a ModelMsg.
func (pm *ModelMsg) SeqNum() int64 {
	return pm.Seq
}

// UpsertMsg creates a model stream upsert message.
func (pm *ModelMsg) UpsertMsg() stream.UpsertMsg {
	return stream.UpsertMsg{
		JSONKey: ModelsUpsertKey,
		Msg:     pm,
	}
}

// DeleteMsg creates a model stream delete message.
func (pm *ModelMsg) DeleteMsg() stream.DeleteMsg {
	deleted := strconv.FormatInt(int64(pm.ID), 10)
	return stream.DeleteMsg{
		Key:     ModelsDeleteKey,
		Deleted: deleted,
	}
}

// ModelSubscriptionSpec is what a user submits to define a Model subscription.
//
// determined:stream-gen source=client
type ModelSubscriptionSpec struct {
	WorkspaceIDs []int `json:"workspace_ids"`
	ModelIDs     []int `json:"Model_ids"`
	Since        int64 `json:"since"`
}

// createFilteredModelIDQuery creates a select query that
// pulls all relevant model ids based on permission scope and
// subscription spec filters.
func createFilteredModelIDQuery(
	globalAccess bool,
	accessScopes []model.AccessScopeID,
	spec ModelSubscriptionSpec,
) *bun.SelectQuery {
	q := db.Bun().NewSelect().
		TableExpr("models m").
		Column("m.id").
		OrderExpr("m.id ASC")

	// add permission scope filter in event of non-global access
	if !globalAccess {
		q = permFilterQuery(q, accessScopes)
	}

	q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		if len(spec.ModelIDs) > 0 {
			q.WhereOr("m.id in (?)", bun.In(spec.ModelIDs))
		}
		if len(spec.WorkspaceIDs) > 0 {
			q.WhereOr("m.workspace_id in (?)", bun.In(spec.WorkspaceIDs))
		}
		return q
	})
	return q
}

// ModelCollectStartupMsgs collects ModelMsg's that were missed prior to startup.
// nolint: dupl
func ModelCollectStartupMsgs(
	ctx context.Context,
	user model.User,
	known string,
	spec ModelSubscriptionSpec,
) (
	[]stream.MarshallableMsg, error,
) {
	var out []stream.MarshallableMsg

	if len(spec.ModelIDs) == 0 && len(spec.WorkspaceIDs) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, stream.DeleteMsg{
			Key:     ModelsDeleteKey,
			Deleted: known,
		})
		return out, nil
	}
	// step 0: get user's permitted access scopes
	accessMap, err := AuthZProvider.Get().GetModelStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}
	globalAccess, accessScopes := getStreamableScopes(accessMap)

	// step 1: figure out what was missing and what has appeared given model subscription
	createQuery := func() *bun.SelectQuery {
		return createFilteredModelIDQuery(
			globalAccess,
			accessScopes,
			spec,
		)
	}
	missing, appeared, err := processQuery(ctx, createQuery, spec.Since, known)
	if err != nil {
		return nil, err
	}

	// step 2: hydrate appeared IDs into full ModelMsgs
	var modelMsgs []*ModelMsg
	if len(appeared) > 0 {
		query := db.Bun().NewSelect().Model(&modelMsgs).Where("id in (?)", bun.In(appeared))
		if !globalAccess {
			query = permFilterQuery(query, accessScopes)
		}
		err := query.Scan(ctx, &modelMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 3: emit deletions and updates to the client
	out = append(out, stream.DeleteMsg{
		Key:     ModelsDeleteKey,
		Deleted: missing,
	})
	for _, msg := range modelMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// ModelMakeFilter creates a ModelMsg filter based on the given ModelSubscriptionSpec.
func ModelMakeFilter(spec *ModelSubscriptionSpec) (func(*ModelMsg) bool, error) {
	// should this filter even run?
	if len(spec.WorkspaceIDs) == 0 && len(spec.ModelIDs) == 0 {
		return nil, errors.Errorf("invalid subscription spec arguments: %v %v", spec.WorkspaceIDs, spec.ModelIDs)
	}

	// create sets based on subscription spec
	workspaceIDs := make(map[int]struct{})
	for _, id := range spec.WorkspaceIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid workspace id: %d", id)
		}
		workspaceIDs[id] = struct{}{}
	}
	modelIDs := make(map[int]struct{})
	for _, id := range spec.ModelIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid model id: %d", id)
		}
		modelIDs[id] = struct{}{}
	}

	// return a closure around our copied maps
	return func(msg *ModelMsg) bool {
		// subscribed to model by this model_id?
		if _, ok := modelIDs[msg.ID]; ok {
			return true
		}
		// subscribed to this model by workspace_id?
		if _, ok := workspaceIDs[msg.WorkspaceID]; ok {
			return true
		}
		return false
	}, nil
}

// ModelMakePermissionFilter returns a function that checks if a ModelMsg
// is in scope of the user permissions.
func ModelMakePermissionFilter(ctx context.Context, user model.User) (func(*ModelMsg) bool, error) {
	accessScopeSet, err := AuthZProvider.Get().GetModelStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}

	switch {
	case accessScopeSet[model.GlobalAccessScopeID]:
		// user has global access for viewing models
		return func(msg *ModelMsg) bool { return true }, nil
	default:
		return func(msg *ModelMsg) bool {
			return accessScopeSet[model.AccessScopeID(msg.WorkspaceID)]
		}, nil
	}
}
