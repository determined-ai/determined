package stream

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

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
	HParams      JsonB            `bun:"hparams" json:"hparams"`

	// warmstart checkpoint id?

	// mutable attributes
	State       model.State `bun:"state" json:"state"`
	StartTime   time.Time   `bun:"start_time" json:"start_time"`
	EndTime     *time.Time  `bun:"end_time" json:"end_time"`
	RunnerState string      `bun:"runner_state" json:"runner_state"`
	Restarts    int         `bun:"restarts" json:"restarts"`
	Tags        JsonB       `bun:"tags" json:"tags"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`

	// total batches?

	upsertCache *websocket.PreparedMessage
	deleteCache *websocket.PreparedMessage
}

func (tm *TrialMsg) SeqNum() int64 {
	return tm.Seq
}

func (tm *TrialMsg) UpsertMsg(upsertFunc stream.UpsertFunc) interface{} {
	wrapper := struct {
		Trial *TrialMsg `json:"trial"`
	}{tm}

	if upsertFunc != nil {
		return upsertFunc(tm)
	}
	return prepareMessageWithCache(wrapper, &tm.upsertCache)
}

func (tm *TrialMsg) DeleteMsg(deleteFunc stream.DeleteFunc) interface{} {
	deleted := strconv.FormatInt(int64(tm.ID), 10)

	if deleteFunc != nil {
		return deleteFunc(TrialsDeleteKey, deleted)
	}
	return newDeletedMsgWithCache(TrialsDeleteKey, deleted, &tm.deleteCache)
}

// TrialSubscriptionSpec is what a user submits to define a trial subscription.
// determined:streamable
type TrialSubscriptionSpec struct {
	TrialIds      []int `json:"trial_ids"`
	ExperimentIds []int `json:"experiment_ids"`
	Since         int64 `json:"since"`
}

// TODO: refactor pls
func TrialCollectStartupMsgs(known string, spec TrialSubscriptionSpec, ctx context.Context, upsertFunc stream.UpsertFunc, deleteFunc stream.DeleteFunc) (
	[]interface{}, error,
) {
	var out []interface{}

	if len(spec.TrialIds) == 0 && len(spec.ExperimentIds) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, newDeletedInterface(TrialsDeleteKey, known, deleteFunc))
		return out, nil
	}

	// step 1: calculate all ids matching this subscription
	q := db.Bun().NewSelect().Table("trials").Column("id").OrderExpr("id ASC")

	// Ignore tmf.Since, because we want appearances, which might not be have seq > spec.Since.
	ws := stream.WhereSince{Since: 0}
	if len(spec.TrialIds) > 0 {
		ws.Include("id in (?)", bun.In(spec.TrialIds))
	}
	if len(spec.ExperimentIds) > 0 {
		ws.Include("experiment_id in (?)", bun.In(spec.ExperimentIds))
	}
	q = ws.Apply(q)

	var exist []int64
	err := q.Scan(ctx, &exist)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		fmt.Printf("error: %v\n", err)
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
		err = db.Bun().NewSelect().Model(&trialMsgs).Where("id in (?)", bun.In(appeared)).Scan(ctx)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			fmt.Printf("error: %v\n", err)
			return nil, err
		}
	}

	// step 4: emit deletions and updates to the client
	out = append(out, newDeletedInterface(TrialsDeleteKey, missing, deleteFunc))
	for _, msg := range trialMsgs {
		out = append(out, msg.UpsertMsg(upsertFunc))
	}
	return out, nil
}

// When a user submits a new TrialSubscriptionSpec, we scrape the database for initial matches.
func TrialCollectSubscriptionModMsgs(addSpec TrialSubscriptionSpec, ctx context.Context, upsertFunc stream.UpsertFunc, deleteFunc stream.DeleteFunc) (
	[]interface{}, error,
) {
	if len(addSpec.TrialIds) == 0 && len(addSpec.ExperimentIds) == 0 {
		return nil, nil
	}
	var trialMsgs []*TrialMsg
	q := db.Bun().NewSelect().Model(&trialMsgs)

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
		fmt.Printf("error: %v\n", err)
		return nil, err
	}

	var out []interface{}
	for _, msg := range trialMsgs {
		out = append(out, msg.UpsertMsg(nil))
	}
	return out, nil
}

type TrialFilterMaker struct {
	TrialIds      map[int]bool
	ExperimentIds map[int]bool
}

func NewTrialFilterMaker() FilterMaker[*TrialMsg, TrialSubscriptionSpec] {
	return &TrialFilterMaker{make(map[int]bool), make(map[int]bool)}
}

func (ts *TrialFilterMaker) AddSpec(spec TrialSubscriptionSpec) {
	for _, id := range spec.TrialIds {
		ts.TrialIds[id] = true
	}
	for _, id := range spec.ExperimentIds {
		ts.ExperimentIds[id] = true
	}
}

func (ts *TrialFilterMaker) DropSpec(spec TrialSubscriptionSpec) {
	for _, id := range spec.TrialIds {
		delete(ts.TrialIds, id)
	}
	for _, id := range spec.ExperimentIds {
		delete(ts.ExperimentIds, id)
	}
}

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
