package stream

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/pkg/model"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/stream"
)

const (
	// CheckpointsDeleteKey specifies the key for delete metrics.
	CheckpointsDeleteKey = "checkpoints_deleted"
	// CheckpointsUpsertKey specifies the key for upsert metrics.
	CheckpointsUpsertKey = "checkpoint"
)

// CheckpointMsg is a stream.Msg. //use checkpoints_v2
// determined:streamable
type CheckpointMsg struct {
	bun.BaseModel `bun:"table:checkpoints_v2"`
	// immutable attributes
	ID           int         `bun:"id,pk" json:"id"`
	UUID         string      `bun:"uuid" json:"uuid"`
	TaskID       string      `bun:"task_id" json:"task_id"`
	AllocationID *string     `bun:"allocation_id" json:"allocation_id"`
	ReportTime   time.Time   `bun:"report_time" json:"report_time"`
	State        model.State `bun:"state" json:"state"`
	Resources    JSONB       `bun:"resources" json:"resources"`
	Metadata     JSONB       `bun:"metadata" json:"metadata"`
	Size         int         `bun:"size" json:"size"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`

	// permission scope
	WorkspaceID int `json:"workspace_id"`

	// TrialID
	TrialID int `json:"trial_id"`

	// ExperimentID
	ExperimentID int `json:"experiment_id"`
}

// SeqNum returns the sequence number of a CheckpointMsg.
func (c *CheckpointMsg) SeqNum() int64 {
	return c.Seq
}

// UpsertMsg creates a Checkpoint upserted prepared message.
func (c *CheckpointMsg) UpsertMsg() stream.UpsertMsg {
	return stream.UpsertMsg{
		JSONKey: CheckpointsUpsertKey,
		Msg:     c,
	}
}

// DeleteMsg creates a Checkpoint deleted prepared message.
func (c *CheckpointMsg) DeleteMsg() stream.DeleteMsg {
	deleted := strconv.FormatInt(int64(c.ID), 10)
	return stream.DeleteMsg{
		Key:     CheckpointsDeleteKey,
		Deleted: deleted,
	}
}

// CheckpointSubscriptionSpec is what a user submits to define a checkpoint subscription.
// determined:streamable
type CheckpointSubscriptionSpec struct {
	TrialIDs      []int `json:"trial_ids"`
	ExperimentIDs []int `json:"experiment_ids"`
	Since         int64 `json:"since"`
}

func getCheckpointMsgsWithWorkspaceID(checkpointMsgs []*CheckpointMsg) *bun.SelectQuery {
	q := db.Bun().NewSelect().Model(&checkpointMsgs).
		Column("id").
		Column("uuid").
		Column("task_id").
		Column("allocation_id").
		Column("report_time").
		Column("state").
		Column("resources").
		Column("metadata").
		Column("size").
		Column("trial_id_task_id.trial_id").
		Column("trials.experiment_id").
		Column("projects.workspace_id").
		Join("JOIN trial_id_task_id ON trial_id_task_id.task_id = checkpoint_msg.task_id").
		Join("JOIN trials ON trial_id_task_id.trial_id = trials.id").
		Join("JOIN experiments ON trials.experiment_id = experiments.id").
		Join("JOIN projects ON experiments.project_id = projects.id")
	return q
}

// CheckpointCollectStartupMsgs collects CheckpointMsg's that were missed prior to startup.
func CheckpointCollectStartupMsgs(
	ctx context.Context,
	user model.User,
	known string,
	spec CheckpointSubscriptionSpec,
) (
	[]stream.PreparableMessage, error,
) {
	var out []stream.PreparableMessage

	if len(spec.TrialIDs) == 0 && len(spec.ExperimentIDs) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, stream.DeleteMsg{
			Key:     CheckpointsDeleteKey,
			Deleted: known,
		})
		return out, nil
	}
	// step 0: get user's permitted access scopes
	accessMap, err := AuthZProvider.Get().GetCheckpointStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}
	var accessScopes []model.AccessScopeID
	for id, isPermitted := range accessMap {
		if isPermitted {
			accessScopes = append(accessScopes, id)
		}
	}

	permFilter := func(q *bun.SelectQuery) *bun.SelectQuery {
		if accessMap[model.GlobalAccessScopeID] {
			return q
		}
		return q.Where("workspace_id in (?)", bun.In(accessScopes))
	}

	// step 1: calculate all ids matching this subscription
	q := db.Bun().
		NewSelect().
		Table("checkpoints_v2").
		Column("checkpoints_v2.id").
		Join("JOIN trial_id_task_id ON trial_id_task_id.task_id = checkpoints_v2.task_id").
		Join("JOIN trials ON trial_id_task_id.trial_id = trials.id").
		Join("JOIN experiments e ON trials.experiment_id = e.id").
		Join("JOIN projects p ON e.project_id = p.id").
		OrderExpr("trials.id ASC")
	q = permFilter(q)

	// Ignore tmf.Since, because we want appearances, which might not be have seq > spec.Since.
	ws := stream.WhereSince{Since: 0}
	if len(spec.TrialIDs) > 0 {
		ws.Include("trials.id in (?)", bun.In(spec.TrialIDs))
	}
	if len(spec.ExperimentIDs) > 0 {
		ws.Include("experiment_id in (?)", bun.In(spec.ExperimentIDs))
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

	// step 3: hydrate appeared IDs into full CheckpointMsgs
	var checkpointMsgs []*CheckpointMsg
	if len(appeared) > 0 {
		query := getCheckpointMsgsWithWorkspaceID(checkpointMsgs).
			Where("trials.id in (?)", bun.In(appeared))
		query = permFilter(query)
		err := query.Scan(ctx, &checkpointMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 4: emit deletions and updates to the client
	out = append(out, stream.DeleteMsg{
		Key:     CheckpointsDeleteKey,
		Deleted: missing,
	})
	for _, msg := range checkpointMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// CheckpointMakeFilter creates a CheckpointMsg filter based on the given CheckpointSubscriptionSpec.
func CheckpointMakeFilter(spec *CheckpointSubscriptionSpec) (func(*CheckpointMsg) bool, error) {
	// Should this filter even run?
	if len(spec.TrialIDs) == 0 && len(spec.ExperimentIDs) == 0 {
		return nil, fmt.Errorf(
			"invalid subscription spec arguments: %v %v",
			spec.TrialIDs, spec.ExperimentIDs,
		)
	}

	// Make a copy of the map, because the filter must run safely off-thread.
	trialIds := make(map[int]struct{})
	for _, id := range spec.TrialIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid trial id: %d", id)
		}
		trialIds[id] = struct{}{}
	}
	experimentIds := make(map[int]struct{})
	for _, id := range spec.ExperimentIDs {
		if id <= 0 {
			return nil, fmt.Errorf("invalid experiment id: %d", id)
		}
		experimentIds[id] = struct{}{}
	}

	// return a closure around our copied map
	return func(msg *CheckpointMsg) bool {
		if _, ok := trialIds[msg.ID]; ok {
			return true
		}
		if _, ok := experimentIds[msg.ExperimentID]; ok {
			return true
		}
		return false
	}, nil
}

// CheckpointMakePermissionFilter returns a function that checks if a CheckpointMsg
// is in scope of the user permissions.
func CheckpointMakePermissionFilter(
	ctx context.Context,
	user model.User,
) (func(*CheckpointMsg) bool, error) {
	accessScopeSet, err := AuthZProvider.Get().GetCheckpointStreamableScopes(
		ctx,
		user,
	)
	if err != nil {
		return nil, err
	}

	switch {
	case accessScopeSet[model.GlobalAccessScopeID]:
		// user has global access for viewing Checkpoints
		return func(msg *CheckpointMsg) bool { return true }, nil
	default:
		return func(msg *CheckpointMsg) bool {
			return accessScopeSet[model.AccessScopeID(msg.WorkspaceID)]
		}, nil
	}
}
