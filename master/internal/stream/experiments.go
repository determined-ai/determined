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
	// ExperimentsUpsertKey specifies the key for upsert experiments.
	ExperimentsUpsertKey = "experiment"
)

// ExperimentMsg is a stream.Msg.
// determined:streamable
type ExperimentMsg struct {
	bun.BaseModel `bun:"table:experiments"`
	// immutable attributes
	ID        int           `bun:"id" json:"id"`
	JobID     model.JobID   `bun:"job_id" json:"job_id"`
	OwnerID   *model.UserID `bun:"owner_id" json:"owner_id"`
	Unmanaged bool          `bun:"unmanaged" json:"unmanaged"`

	// mutable attributes
	ProjectID int         `bun:"project_id" json:"project_id"`
	State     model.State `bun:"state" json:"state"`
	Archived  bool        `bun:"archived" json:"archived"`
	Progress  *float64    `bun:"progress" json:"progress"`
	StartTime time.Time   `bun:"start_time" json:"start_time"`
	EndTime   *time.Time  `bun:"end_time" json:"end_time"`
	Notes     string      `bun:"notes" json:"notes"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`

	// permission scope
	WorkspaceID int `json:"-"`
}

// SeqNum returns the sequence number of an experiment message.
func (e *ExperimentMsg) SeqNum() int64 {
	return e.Seq
}

// UpsertMsg creates an Experiment upserted prepared message.
func (e *ExperimentMsg) UpsertMsg() stream.UpsertMsg {
	return stream.UpsertMsg{
		JSONKey: ExperimentsUpsertKey,
		Msg:     e,
	}
}

// DeleteMsg creates an Experiment deleted prepared message.
func (e *ExperimentMsg) DeleteMsg() stream.DeleteMsg {
	deleted := strconv.FormatInt(int64(e.ID), 10)
	return stream.DeleteMsg{
		Key:     ExperimentsDeleteKey,
		Deleted: deleted,
	}
}

// ExperimentSubscriptionSpec is what a user submits to define an experiment subscription.
// determined:streamable
type ExperimentSubscriptionSpec struct {
	ExperimentIds []int `json:"experiment_ids"`
	Since         int64 `json:"since"`
}

func getExperimentMsgsWithWorkspaceID(expMsgs []*ExperimentMsg) *bun.SelectQuery {
	q := db.Bun().NewSelect().Model(&expMsgs).
		Column("id").
		Column("job_id").
		Column("owner_id").
		Column("project_id").
		Column("unmanaged").
		Column("state").
		Column("archived").
		Column("progress").
		Column("start_time").
		Column("end_time").
		Column("notes").
		Column("projects.workspace_id").
		Join("JOIN projects ON experiment_msg.project_id = projects.id")
	return q
}

// ExperimentCollectStartupMsgs collects ExperimentMsg's that were missed prior to startup.
func ExperimentCollectStartupMsgs(
	ctx context.Context,
	user model.User,
	known string,
	spec ExperimentSubscriptionSpec,
) (
	[]stream.PreparableMessage, error,
) {
	var out []stream.PreparableMessage

	if len(spec.ExperimentIds) == 0 {
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
	var accessScopes []model.AccessScopeID
	for id, isPermitted := range accessMap {
		if isPermitted {
			accessScopes = append(accessScopes, id)
		}
	}

	// step 1: calculate all ids matching this subscription
	q := db.Bun().NewSelect().
		TableExpr("experiments e").
		Column("e.id").
		Join("JOIN projects p ON e.project_id = p.id").
		OrderExpr("e.id ASC")

	q = permFilter(q, accessMap, accessScopes)

	// Ignore tmf.Since, because we want appearances, which might not be have seq > spec.Since.
	ws := stream.WhereSince{Since: 0}
	if len(spec.ExperimentIds) > 0 {
		ws.Include("e.id in (?)", bun.In(spec.ExperimentIds))
	}
	q = ws.Apply(q)

	var exist []int64
	err = q.Scan(ctx, &exist)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("error: %v\n", err)
		return nil, err
	}

	// step 2: figure out what was missing and what has appeared
	missing, appeared, err := stream.ProcessKnown(known, exist)
	if err != nil {
		return nil, err
	}

	// step 3: hydrate appeared IDs into full ExperimentMsgs
	var expMsgs []*ExperimentMsg
	if len(appeared) > 0 {
		query := getExperimentMsgsWithWorkspaceID(expMsgs).
			Where("experiment_msg.id in (?)", bun.In(appeared))
		query = permFilter(query, accessMap, accessScopes)
		err := query.Scan(ctx, &expMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 4: emit deletions and updates to the client
	out = append(out, stream.DeleteMsg{
		Key:     ExperimentsDeleteKey,
		Deleted: missing,
	})
	for _, msg := range expMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// ExperimentMakeFilter returns a function that determines if a ExperimentMsg based on
// the ExperimentFilterMaker's spec.
func ExperimentMakeFilter(spec *ExperimentSubscriptionSpec) (func(msg *ExperimentMsg) bool, error) {
	// Should this filter even run?
	if len(spec.ExperimentIds) == 0 {
		return nil, fmt.Errorf("invalid subscription spec arguments: %v", spec.ExperimentIds)
	}

	// Make a copy of the maps, because the filter must run safely off-thread.
	experimentIds := make(map[int]struct{})
	for _, id := range spec.ExperimentIds {
		if id <= 0 {
			return nil, fmt.Errorf("invalid experiment id: %d", id)
		}
		experimentIds[id] = struct{}{}
	}

	// return a closure around our copied maps
	return func(msg *ExperimentMsg) bool {
		if _, ok := experimentIds[msg.ID]; ok {
			return true
		}
		return false
	}, nil
}

// ExperimentMakePermissionFilter returns a function that checks if a ExperimentMsg
// is in scope of the user permissions.
func ExperimentMakePermissionFilter(ctx context.Context, user model.User) (func(msg *ExperimentMsg) bool, error) {
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
			return accessScopeSet[model.AccessScopeID(msg.WorkspaceID)]
		}, nil
	}
}
