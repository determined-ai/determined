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
	// ExperimentsDeleteKey specifies the key for delete experiments.
	ExperimentsDeleteKey = "experiments_deleted"
	// ExperimentUpsertKey specifies the key for upsert experiments.
	ExperimentUpsertKey = "experiment"
	// experimentChannel specifies the channel to listen to experiment events.
	experimentChannel = "stream_experiment_chan"
)

// ExperimentMsg is a stream.Msg.
//
// determined:stream-gen source=server delete_msg=ProjectsDeleted
type ExperimentMsg struct {
	bun.BaseModel `bun:"table:experiments"`

	// immutable attributes
	ID int `bun:"id,pk" json:"id"`

	// mutable attributes
	State                model.State `bun:"state" json:"state"`
	Config               JSONB       `bun:"config,type:jsonb" json:"config"`
	ModelDefinition      []byte      `bun:"model_definition" json:"model_definition"`
	StartTime            time.Time   `bun:"start_time" json:"start_time"`
	EndTime              time.Time   `bun:"end_time" json:"end_time"`
	Archived             bool        `bun:"archived" json:"archived"`
	ParentID             int         `bun:"parent_id" json:"parent_id"`
	OwnerID              int         `bun:"owner_id" json:"owner_id"`
	Progress             float64     `bun:"progress" json:"progress"`
	OriginalConfig       string      `bun:"original_config" json:"original_config"`
	Notes                string      `bun:"notes" json:"notes"`
	JobID                string      `bun:"job_id" json:"job_id"`
	ProjectID            int         `bun:"project_id" json:"project_id"`
	CheckpointSize       int64       `bun:"checkpoint_size" json:"checkpoint_size"`
	CheckpointCount      int         `bun:"checkpoint_count" json:"checkpoint_count"`
	BestTrialID          int         `bun:"best_trial_id" json:"best_trial_id"`
	Unmanaged            bool        `bun:"unmanaged" json:"unmanaged"`
	ExternalExperimentID string      `bun:"external_experiment_id" json:"external_experiment_id"`
	WorkspaceID          string      `json:"workspace_id"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`
}

// SeqNum gets the SeqNum from a ExperimentMsg.
func (em *ExperimentMsg) SeqNum() int64 {
	return em.Seq
}

// GetID gets the ID from a ExperimentMsg.
func (em *ExperimentMsg) GetID() int {
	return em.ID
}

// UpsertMsg creates a Experiment stream upsert message.
func (em *ExperimentMsg) UpsertMsg() stream.UpsertMsg {
	return stream.UpsertMsg{
		JSONKey: ExperimentUpsertKey,
		Msg:     em,
	}
}

// DeleteMsg creates a Experiment stream delete message.
func (em *ExperimentMsg) DeleteMsg() stream.DeleteMsg {
	deleted := strconv.FormatInt(int64(em.ID), 10)
	return stream.DeleteMsg{
		Key:     ExperimentsDeleteKey,
		Deleted: deleted,
	}
}

// ExperimentSubscriptionSpec is what a user submits to define a experiment subscription.
//
// determined:stream-gen source=client
type ExperimentSubscriptionSpec struct {
	ExperimentIDs []int `json:"experiment_ids"`
	ProjectIDs    []int `json:"project_ids"`
	Since         int64 `json:"since"`
}

func experimentPermFilterQuery(q *bun.SelectQuery, accessScopes []model.AccessScopeID,
) *bun.SelectQuery {
	return q.Join("JOIN projects ON projects.id = project_id").Where("workspace_id in (?)", bun.In(accessScopes))
}

// createFilteredExperimentIDQuery creates a select query that
// pulls all relevant experiment ids based on permission scope and
// subscription spec filters.
func createFilteredExperimentIDQuery(
	globalAccess bool,
	accessScopes []model.AccessScopeID,
	spec ExperimentSubscriptionSpec,
) *bun.SelectQuery {
	q := db.Bun().NewSelect().
		TableExpr("experiments e").
		Column("e.id").
		OrderExpr("e.id ASC")

	// add permission scope filter in event of non-global access
	if !globalAccess {
		q = experimentPermFilterQuery(q, accessScopes)
	}

	q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
		if len(spec.ExperimentIDs) > 0 {
			q.WhereOr("e.id in (?)", bun.In(spec.ExperimentIDs))
		}
		if len(spec.ProjectIDs) > 0 {
			q.WhereOr("e.project_id in (?)", bun.In(spec.ProjectIDs))
		}
		return q
	})
	return q
}

// ExperimentCollectStartupMsgs collects ExperimentMsg's that were missed prior to startup.
// nolint: dupl
func ExperimentCollectStartupMsgs(
	ctx context.Context,
	user model.User,
	known string,
	spec ExperimentSubscriptionSpec,
) (
	[]stream.MarshallableMsg, error,
) {
	var out []stream.MarshallableMsg

	if len(spec.ExperimentIDs) == 0 && len(spec.ProjectIDs) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, stream.DeleteMsg{
			Key:     ExperimentsDeleteKey,
			Deleted: known,
		})
		return out, nil
	}
	// step 0: get user's permitted access scopes
	accessMap, err := AuthZProvider.Get().GetExperimentStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}
	globalAccess, accessScopes := getStreamableScopes(accessMap)

	// step 1: calculate all ids matching this subscription
	createQuery := func() *bun.SelectQuery {
		return createFilteredExperimentIDQuery(
			globalAccess,
			accessScopes,
			spec,
		)
	}
	missing, appeared, err := processQuery(ctx, createQuery, spec.Since, known, "e")
	if err != nil {
		return nil, fmt.Errorf("processing known: %s", err.Error())
	}

	// step 2: hydrate appeared IDs into full ExperimentMsgs
	var expMsgs []*ExperimentMsg
	if len(appeared) > 0 {
		query := db.Bun().NewSelect().Model(&expMsgs).Where("id in (?)", bun.In(appeared))
		if !globalAccess {
			query = permFilterQuery(query, accessScopes)
		}
		err := query.Scan(ctx, &expMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 3: emit deletions and updates to the client
	out = append(out, stream.DeleteMsg{
		Key:     ExperimentsDeleteKey,
		Deleted: missing,
	})
	for _, msg := range expMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// ExperimentMakeFilter creates a ExperimentMsg filter based on the given ExperimentSubscriptionSpec.
func ExperimentMakeFilter(spec *ExperimentSubscriptionSpec) (func(*ExperimentMsg) bool, error) {
	// should this filter even run?
	if len(spec.ExperimentIDs) == 0 && len(spec.ProjectIDs) == 0 {
		return nil, errors.Errorf("invalid subscription spec arguments: %v %v", spec.ExperimentIDs, spec.ProjectIDs)
	}

	// create sets based on subscription spec
	workspaceIDs := make(map[int]struct{})
	for _, id := range spec.ExperimentIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid experiment id: %d", id)
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
	return func(msg *ExperimentMsg) bool {
		// subscribed to experiment by this project_id?
		if _, ok := projectIDs[msg.ID]; ok {
			return true
		}
		// subscribed to this experiment by workspace_id?
		if _, ok := workspaceIDs[msg.ProjectID]; ok {
			return true
		}
		return false
	}, nil
}

// ExperimentMakePermissionFilter returns a function that checks if a ExperimentMsg
// is in scope of the user permissions.
func ExperimentMakePermissionFilter(ctx context.Context, user model.User) (func(*ExperimentMsg) bool, error) {
	accessScopeSet, err := AuthZProvider.Get().GetExperimentStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}

	switch {
	case accessScopeSet[model.GlobalAccessScopeID]:
		// user has global access for viewing experiments
		return func(msg *ExperimentMsg) bool { return true }, nil
	default:
		return func(msg *ExperimentMsg) bool {
			return accessScopeSet[model.AccessScopeID(msg.ID)]
		}, nil
	}
}

// ExperimentMakeHydrator returns a function that gets properties of a experiment by
// its id.
func ExperimentMakeHydrator() func(int) (*ExperimentMsg, error) {
	return func(ID int) (*ExperimentMsg, error) {
		var expMsg ExperimentMsg
		query := db.Bun().NewSelect().Model(&expMsg).Where("id = ?", ID)
		err := query.Scan(context.Background(), &expMsg)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error in experiment hydrator: %v\n", err)
			return nil, err
		}
		return &expMsg, nil
	}
}
