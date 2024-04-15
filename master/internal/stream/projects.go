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
	"github.com/uptrace/bun/extra/bundebug"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

const (
	// ProjectsDeleteKey specifies the key for delete projects.
	ProjectsDeleteKey = "projects_deleted"
	// ProjectsUpsertKey specifies the key for upsert projects.
	ProjectsUpsertKey = "project"
	// projectChannel specifies the channel to listen to project events.
	projectChannel = "stream_project_chan"
)

// JSONB is the golang equivalent of the postgres jsonb column type.
type JSONB interface{}

// ProjectMsg is a stream.Msg.
//
// determined:stream-gen source=server delete_msg=ProjectsDeleted
type ProjectMsg struct {
	bun.BaseModel `bun:"table:projects"`

	// immutable attributes
	ID int `bun:"id,pk" json:"id"`

	// mutable attributes
	Name        string               `bun:"name" json:"name"`
	Description string               `bun:"description" json:"description"`
	Archived    bool                 `bun:"archived" json:"archived"`
	CreatedAt   time.Time            `bun:"created_at" json:"created_at"`
	Notes       JSONB                `bun:"notes,type:jsonb" json:"notes"`
	WorkspaceID int                  `bun:"workspace_id" json:"workspace_id"`
	UserID      int                  `bun:"user_id" json:"user_id"`
	Immutable   bool                 `bun:"immutable" json:"immutable"`
	State       model.WorkspaceState `bun:"state" json:"state"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`
}

// SeqNum gets the SeqNum from a ProjectMsg.
func (pm ProjectMsg) SeqNum() int64 {
	return pm.Seq
}

// GetID gets the ID from a ProjectMsg.
func (pm ProjectMsg) GetID() int {
	return pm.ID
}

// UpsertMsg creates a Project stream upsert message.
func (pm ProjectMsg) UpsertMsg() stream.UpsertMsg {
	// hydrate
	// db.Bun().AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))

	// db.Bun().AddQueryHook(bundebug.NewQueryHook(bundebug.WithEnabled(false)))
	return stream.UpsertMsg{
		JSONKey: ProjectsUpsertKey,
		// This has to be & because project filter and permission filter need *ProjectMsg
		Msg: &pm,
	}
}

// DeleteMsg creates a Project stream delete message.
func (pm ProjectMsg) DeleteMsg() stream.DeleteMsg {
	deleted := strconv.FormatInt(int64(pm.ID), 10)
	return stream.DeleteMsg{
		Key:     ProjectsDeleteKey,
		Deleted: deleted,
	}
}

// ProjectSubscriptionSpec is what a user submits to define a project subscription.
//
// determined:stream-gen source=client
type ProjectSubscriptionSpec struct {
	WorkspaceIDs []int `json:"workspace_ids"`
	ProjectIDs   []int `json:"project_ids"`
	Since        int64 `json:"since"`
}

// createFilteredProjectIDQuery creates a select query that
// pulls all relevant project ids based on permission scope and
// subscription spec filters.
func createFilteredProjectIDQuery(
	globalAccess bool,
	accessScopes []model.AccessScopeID,
	spec ProjectSubscriptionSpec,
) *bun.SelectQuery {
	q := db.Bun().NewSelect().
		TableExpr("projects p").
		Column("p.id").
		OrderExpr("p.id ASC")

	// add permission scope filter in event of non-global access
	if !globalAccess {
		q = permFilterQuery(q, accessScopes)
	}

	q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		if len(spec.ProjectIDs) > 0 {
			q.WhereOr("p.id in (?)", bun.In(spec.ProjectIDs))
		}
		if len(spec.WorkspaceIDs) > 0 {
			q.WhereOr("p.workspace_id in (?)", bun.In(spec.WorkspaceIDs))
		}
		return q
	})
	return q
}

// ProjectCollectStartupMsgs collects ProjectMsg's that were missed prior to startup.
// nolint: dupl
func ProjectCollectStartupMsgs(
	ctx context.Context,
	user model.User,
	known string,
	spec ProjectSubscriptionSpec,
) (
	[]stream.MarshallableMsg, error,
) {
	var out []stream.MarshallableMsg

	if len(spec.ProjectIDs) == 0 && len(spec.WorkspaceIDs) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, stream.DeleteMsg{
			Key:     ProjectsDeleteKey,
			Deleted: known,
		})
		return out, nil
	}
	// step 0: get user's permitted access scopes
	accessMap, err := AuthZProvider.Get().GetProjectStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}
	globalAccess, accessScopes := getStreamableScopes(accessMap)

	// step 1: calculate all ids matching this subscription
	createQuery := func() *bun.SelectQuery {
		return createFilteredProjectIDQuery(
			globalAccess,
			accessScopes,
			spec,
		)
	}
	missing, appeared, err := processQuery(ctx, createQuery, spec.Since, known, "p")
	if err != nil {
		return nil, fmt.Errorf("processing known: %s", err.Error())
	}
	// fmt.Printf("deleted Entities: %+v\n", missing)

	// step 2: hydrate appeared IDs into full ProjectMsgs
	db.Bun().AddQueryHook(bundebug.NewQueryHook(bundebug.WithVerbose(true)))
	var projMsgs []*ProjectMsg
	if len(appeared) > 0 {
		query := db.Bun().NewSelect().Model(&projMsgs).Where("project_msg.id in (?)", bun.In(appeared))
		if !globalAccess {
			query = permFilterQuery(query, accessScopes)
		}
		err := query.Scan(ctx, &projMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}
	db.Bun().AddQueryHook(bundebug.NewQueryHook(bundebug.WithEnabled(false)))

	// step 3: emit deletions and updates to the client
	out = append(out, stream.DeleteMsg{
		Key:     ProjectsDeleteKey,
		Deleted: missing,
	})
	for _, msg := range projMsgs {
		upsertMsg := msg.UpsertMsg()
		out = append(out, upsertMsg)
	}
	return out, nil
}

// ProjectMakeFilter creates a ProjectMsg filter based on the given ProjectSubscriptionSpec.
func ProjectMakeFilter(spec *ProjectSubscriptionSpec) (func(*ProjectMsg) bool, error) {
	// should this filter even run?
	if len(spec.WorkspaceIDs) == 0 && len(spec.ProjectIDs) == 0 {
		return nil, errors.Errorf("invalid subscription spec arguments: %v %v", spec.WorkspaceIDs, spec.ProjectIDs)
	}

	// create sets based on subscription spec
	workspaceIDs := make(map[int]struct{})
	for _, id := range spec.WorkspaceIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid workspace id: %d", id)
		}
		workspaceIDs[id] = struct{}{}
	}
	projectIDs := make(map[int]struct{})
	for _, id := range spec.ProjectIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid project id: %d", id)
		}
		projectIDs[id] = struct{}{}
	}

	// return a closure around our copied maps
	return func(msg *ProjectMsg) bool {
		// subscribed to project by this project_id?
		if _, ok := projectIDs[msg.ID]; ok {
			return true
		}
		// subscribed to this project by workspace_id?
		if _, ok := workspaceIDs[msg.WorkspaceID]; ok {
			return true
		}
		return false
	}, nil
}

// ProjectMakePermissionFilter returns a function that checks if a ProjectMsg
// is in scope of the user permissions.
func ProjectMakePermissionFilter(ctx context.Context, user model.User) (func(*ProjectMsg) bool, error) {
	accessScopeSet, err := AuthZProvider.Get().GetProjectStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}

	switch {
	case accessScopeSet[model.GlobalAccessScopeID]:
		// user has global access for viewing projects
		return func(msg *ProjectMsg) bool { return true }, nil
	default:
		return func(msg *ProjectMsg) bool {
			return accessScopeSet[model.AccessScopeID(msg.WorkspaceID)]
		}, nil
	}
}

// ProjectMakeFilter creates a ProjectMsg filter based on the given ProjectSubscriptionSpec.
func ProjectHydrateUpsertMsg() func(int) (*ProjectMsg, error) {
	return func(ID int) (*ProjectMsg, error) {
		var projMsg ProjectMsg
		query := db.Bun().NewSelect().Model(&projMsg).Where("project_msg.id = ?", ID)
		err := query.Scan(context.Background(), &projMsg)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
		return &projMsg, nil
	}
}
