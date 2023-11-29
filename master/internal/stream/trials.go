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
	// TrialsDeleteKey specifies the key for delete trials.
	TrialsDeleteKey = "trials_deleted"
	// TrialsUpsertKey specifies the key for upsert trials.
	TrialsUpsertKey = "trial"
)

// TrialMsg is a stream.Msg.
type TrialMsg struct {
	bun.BaseModel `bun:"table:trials"`

	// immutable attributes
	ID           int              `bun:"id,pk" json:"id"`
	ExperimentID int              `bun:"experiment_id" json:"experiment_id"`
	RequestID    *model.RequestID `bun:"request_id" json:"request_id"`
	Seed         int64            `bun:"seed" json:"seed"`
	HParams      JSONB            `bun:"hparams" json:"hparams"`

	// warmstart checkpoint id?

	// mutable attributes
	State       model.State `bun:"state" json:"state"`
	StartTime   time.Time   `bun:"start_time" json:"start_time"`
	EndTime     *time.Time  `bun:"end_time" json:"end_time"`
	RunnerState string      `bun:"runner_state" json:"runner_state"`
	Restarts    int         `bun:"restarts" json:"restarts"`
	Tags        JSONB       `bun:"tags" json:"tags"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`

	// permission scope
	WorkspaceID int `json:"-"`
}

// SeqNum gets the SeqNum from a TrialMsg.
func (tm *TrialMsg) SeqNum() int64 {
	return tm.Seq
}

// UpsertMsg creates a Trial stream upsert message.
func (tm *TrialMsg) UpsertMsg() stream.UpsertMsg {
	return stream.UpsertMsg{
		JSONKey: TrialsUpsertKey,
		Msg:     tm,
	}
}

// DeleteMsg creates a Trial stream delete message.
func (tm *TrialMsg) DeleteMsg() stream.DeleteMsg {
	deleted := strconv.FormatInt(int64(tm.ID), 10)
	return stream.DeleteMsg{
		Key:     TrialsDeleteKey,
		Deleted: deleted,
	}
}

// TrialSubscriptionSpec is what a user submits to define a trial subscription.
type TrialSubscriptionSpec struct {
	TrialIds      []int `json:"trial_ids"`
	ExperimentIds []int `json:"experiment_ids"`
	Since         int64 `json:"since"`
}

func getTrialMsgsWithWorkspaceID(trialMsgs []*TrialMsg) *bun.SelectQuery {
	q := db.Bun().NewSelect().Model(&trialMsgs).
		Column("id").
		Column("experiment_id").
		Column("request_id").
		Column("seed").
		Column("hparams").
		Column("state").
		Column("start_time").
		Column("end_time").
		Column("runner_state").
		Column("restarts").
		Column("tags").
		Column("seq").
		Column("projects.workspace_id").
		Join("JOIN experiments ON trial_msg.experiment_id = experiments.id").
		Join("JOIN projects ON experiments.project_id = projects.id")
	return q
}

// TrialCollectStartupMsgs collects TrialMsg's that were missed prior to startup.
func TrialCollectStartupMsgs(
	ctx context.Context,
	user model.User,
	known string,
	spec TrialSubscriptionSpec,
) (
	[]stream.PreparableMessage, error,
) {
	var out []stream.PreparableMessage

	if len(spec.TrialIds) == 0 && len(spec.ExperimentIds) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, stream.DeleteMsg{
			Key:     TrialsDeleteKey,
			Deleted: known,
		})
		return out, nil
	}
	// step 0: get user's permitted access scopes
	accessMap, err := AuthZProvider.Get().GetTrialStreamableScopes(ctx, user)
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
	q := db.Bun().
		NewSelect().
		Table("trials").
		Column("trials.id").
		Join("JOIN experiments e ON trials.experiment_id = e.id").
		Join("JOIN projects p ON e.project_id = p.id").
		OrderExpr("trials.id ASC")
	q = permFilter(q, accessMap, accessScopes)

	// Ignore tmf.Since, because we want appearances, which might not be have seq > spec.Since.
	ws := stream.WhereSince{Since: 0}
	if len(spec.TrialIds) > 0 {
		ws.Include("trials.id in (?)", bun.In(spec.TrialIds))
	}
	if len(spec.ExperimentIds) > 0 {
		ws.Include("experiment_id in (?)", bun.In(spec.ExperimentIds))
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

	// step 3: hydrate appeared IDs into full TrialMsgs
	var trialMsgs []*TrialMsg
	if len(appeared) > 0 {
		query := getTrialMsgsWithWorkspaceID(trialMsgs).
			Where("trial_msg.id in (?)", bun.In(appeared))
		query = permFilter(query, accessMap, accessScopes)
		err := query.Scan(ctx, &trialMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 4: emit deletions and updates to the client
	out = append(out, stream.DeleteMsg{
		Key:     TrialsDeleteKey,
		Deleted: missing,
	})
	for _, msg := range trialMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// TrialMakeFilter creates a TrialMsg filter based on the given TrialSubscriptionSpec.
func TrialMakeFilter(spec *TrialSubscriptionSpec) (func(*TrialMsg) bool, error) {
	// should this filter even run?
	if len(spec.TrialIds) == 0 && len(spec.ExperimentIds) == 0 {
		return nil, errors.Errorf("invalid subscription spec arguments: %v %v", spec.TrialIds, spec.ExperimentIds)
	}
	// create sets based on subscription spec
	trialIds := make(map[int]struct{})
	for _, id := range spec.TrialIds {
		if id <= 0 {
			return nil, fmt.Errorf("invalid trial id: %d", id)
		}
		trialIds[id] = struct{}{}
	}
	experimentIds := make(map[int]struct{})
	for _, id := range spec.ExperimentIds {
		if id <= 0 {
			return nil, fmt.Errorf("invalid experiment id: %d", id)
		}
		experimentIds[id] = struct{}{}
	}

	// return a closure around our copied maps
	return func(msg *TrialMsg) bool {
		if _, ok := trialIds[msg.ID]; ok {
			return true
		}
		if _, ok := experimentIds[msg.ExperimentID]; ok {
			return true
		}
		return false
	}, nil
}

// TrialMakePermissionFilter returns a function that checks if a TrialMsg
// is in scope of the user permissions.
func TrialMakePermissionFilter(ctx context.Context, user model.User) (func(*TrialMsg) bool, error) {
	accessScopeSet, err := AuthZProvider.Get().GetTrialStreamableScopes(ctx, user)
	if err != nil {
		return nil, err
	}

	switch {
	case accessScopeSet[model.GlobalAccessScopeID]:
		// user has global access for viewing trials
		return func(msg *TrialMsg) bool { return true }, nil
	default:
		return func(msg *TrialMsg) bool {
			return accessScopeSet[model.AccessScopeID(msg.WorkspaceID)]
		}, nil
	}
}
