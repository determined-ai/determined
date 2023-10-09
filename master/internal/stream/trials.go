package stream

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

// TrialsDeleteKey specifies the key for delete trials.
const TrialsDeleteKey = "trials_deleted"

// TrialMsg is a stream.Msg.
// determined:streamable
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

	upsertCache *websocket.PreparedMessage
	deleteCache *websocket.PreparedMessage
}

// SeqNum gets the SeqNum from a TrialMsg.
func (tm *TrialMsg) SeqNum() int64 {
	return tm.Seq
}

// UpsertMsg creates a Trial upserted prepared message.
func (tm *TrialMsg) UpsertMsg() *websocket.PreparedMessage {
	wrapper := struct {
		Trial *TrialMsg `json:"trial"`
	}{tm}
	return prepareMessageWithCache(wrapper, &tm.upsertCache)
}

// DeleteMsg creates a Trial deleted prepared message.
func (tm *TrialMsg) DeleteMsg() *websocket.PreparedMessage {
	deleted := strconv.FormatInt(int64(tm.ID), 10)
	return newDeletedMsgWithCache(TrialsDeleteKey, deleted, &tm.deleteCache)
}

// TrialSubscriptionSpec is what a user submits to define a trial subscription.
// determined:streamable
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
	[]*websocket.PreparedMessage, error,
) {
	var out []*websocket.PreparedMessage

	if len(spec.TrialIds) == 0 && len(spec.ExperimentIds) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, newDeletedMsg(TrialsDeleteKey, known))
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

	permFilter := func(q *bun.SelectQuery) *bun.SelectQuery {
		if accessMap[model.GlobalAccessScopeID] {
			return q
		}
		return q.Where("workspace_id in (?)", bun.In(accessScopes))
	}

	// step 1: calculate all ids matching this subscription
	q := db.Bun().
		NewSelect().
		Table("trials").
		Column("trials.id").
		Join("JOIN experiments e ON trials.experiment_id = e.id").
		Join("JOIN projects p ON e.project_id = p.id").
		OrderExpr("trials.id ASC")
	q = permFilter(q)

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
		query = permFilter(query)
		err := query.Scan(ctx, &trialMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 4: emit deletions and updates to the client
	out = append(out, newDeletedMsg(TrialsDeleteKey, missing))
	for _, msg := range trialMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// TrialCollectSubscriptionModMsgs scrapes the database when a
// user submits a new TrialSubscriptionSpec for initial matches.
func TrialCollectSubscriptionModMsgs(ctx context.Context, addSpec TrialSubscriptionSpec) (
	[]*websocket.PreparedMessage, error,
) {
	if len(addSpec.TrialIds) == 0 && len(addSpec.ExperimentIds) == 0 {
		return nil, nil
	}
	var trialMsgs []*TrialMsg
	q := getTrialMsgsWithWorkspaceID(trialMsgs)

	// Use WhereSince to build a complex WHERE clause.
	ws := stream.WhereSince{Since: addSpec.Since}
	if len(addSpec.TrialIds) > 0 {
		ws.Include("id in (?)", bun.In(addSpec.TrialIds))
	}
	if len(addSpec.ExperimentIds) > 0 {
		ws.Include("experiment_id in (?)", bun.In(addSpec.ExperimentIds))
	}
	q = ws.Apply(q)

	err := q.Scan(ctx)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("error: %v\n", err)
		return nil, err
	}

	var out []*websocket.PreparedMessage
	for _, msg := range trialMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// TrialFilterMaker tracks the trial and experiment id's that are to be filtered for.
type TrialFilterMaker struct {
	TrialIds      map[int]bool
	ExperimentIds map[int]bool
}

// NewTrialFilterMaker creates a new FilterMaker.
func NewTrialFilterMaker() FilterMaker[*TrialMsg, TrialSubscriptionSpec] {
	return &TrialFilterMaker{make(map[int]bool), make(map[int]bool)}
}

// AddSpec adds TrialIds and ExperimentIds specified in TrialSubscriptionSpec.
func (ts *TrialFilterMaker) AddSpec(spec TrialSubscriptionSpec) {
	for _, id := range spec.TrialIds {
		ts.TrialIds[id] = true
	}
	for _, id := range spec.ExperimentIds {
		ts.ExperimentIds[id] = true
	}
}

// DropSpec removes TrialIds and ExperimentIds specified in TrialSubscriptionSpec.
func (ts *TrialFilterMaker) DropSpec(spec TrialSubscriptionSpec) {
	for _, id := range spec.TrialIds {
		delete(ts.TrialIds, id)
	}
	for _, id := range spec.ExperimentIds {
		delete(ts.ExperimentIds, id)
	}
}

// MakeFilter returns a function that determines if a TrialMsg based on
// the TrialFilterMaker's spec.
func (ts *TrialFilterMaker) MakeFilter() func(*TrialMsg) bool {
	// Should this filter even run?
	if len(ts.TrialIds) == 0 && len(ts.ExperimentIds) == 0 {
		return nil
	}

	// Make a copy of the maps, because the filter must run safely off-thread.
	trialIds := make(map[int]bool)
	experimentIds := make(map[int]bool)
	for id := range ts.TrialIds {
		trialIds[id] = true
	}
	for id := range ts.ExperimentIds {
		experimentIds[id] = true
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
	}
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
