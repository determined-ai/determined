package stream

import (
	"context"
	"database/sql"
	"time"
	"fmt"
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	log "github.com/sirupsen/logrus"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
)


// TrialMsg is a stream.Event.
type TrialMsg struct {
	bun.BaseModel `bun:"table:trials"`

	// immutable attributes
	ID int                      `bun:"id,pk"`
	TaskID model.TaskID         `bun:"task_id"`
	ExperimentID int            `bun:"experiment_id"`
	RequestID *model.RequestID  `bun:"request_id"`
	Seed int64                  `bun:"seed"`
	HParams JsonB               `bun:"hparams"`

	// warmstart checkpoint id?

	// mutable attributes
	State model.State           `bun:"state"`
	StartTime time.Time         `bun:"start_time"`
	EndTime *time.Time          `bun:"end_time"`
	RunnerState string          `bun:"runner_state"`
	Restarts int                `bun:"restarts"`
	Tags JsonB                  `bun:"tags"`

	// metadata
	Seq int64                   `bun:"seq"`

	// total batches?

	cache *websocket.PreparedMessage
}

func (tm *TrialMsg) SeqNum() int64 {
	return tm.Seq
}

func (tm *TrialMsg) PreparedMessage() *websocket.PreparedMessage {
	return prepareMessageWithCache(tm, &tm.cache)
}

// scan for updates to the trials table
func newTrialMsgs(since int64, ctx context.Context) (int64, []*TrialMsg, error) {
	var trialMsgs []*TrialMsg
	err := db.Bun().NewSelect().Model(&trialMsgs).Where("seq > ?", since).Scan(ctx)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		fmt.Printf("error: %v\n", err)
		return since, nil, err
	}

	newSince := since
	for _, tm := range trialMsgs {
		if tm.Seq > newSince {
			newSince = tm.Seq
		}
	}

	return newSince, trialMsgs, nil
}

// TrialFilterMod is what a user submits to define a trial subscription.
type TrialFilterMod struct {
	TrialIds      []int  `json:"trial_ids"`
	ExperimentIds []int  `json:"experiment_ids"`
	Since         int64  `json:"since"`
}

type TrialsDeletedMsg struct {
	TrialsDeleted string `json:"trials_deleted"`
}

func (tdm TrialsDeletedMsg) PreparedMessage() *websocket.PreparedMessage {
	jbytes, err := json.Marshal(tdm)
	if err != nil {
		log.Errorf("error marshaling message for streaming: %v", err.Error())
		return nil
	}
	msg, err := websocket.NewPreparedMessage(websocket.BinaryMessage, jbytes)
	if err != nil {
		log.Errorf("error marshaling message for streaming: %v", err.Error())
		return nil
	}
	return msg
}

func (tfm TrialFilterMod) Startup(known string, ctx context.Context) (
	[]*websocket.PreparedMessage, error,
) {
	var out []*websocket.PreparedMessage

	if len(tfm.TrialIds) == 0 && len(tfm.ExperimentIds) == 0 {
		// empty subscription: everything known should be returned as deleted
		tdm := TrialsDeletedMsg{known}
		out = append(out, tdm.PreparedMessage())
		return out, nil
	}

	// step 1: calculate all ids matching this subscription
	q := db.Bun().NewSelect().Table("trials").Column("id")

	// Ignore tmf.Since, because we want appearances, which might not be have seq > tfm.Since.
	ws := WhereSince{Since: 0}
	if len(tfm.TrialIds) > 0 {
		ws.Include("id in (?)", bun.In(tfm.TrialIds))
	}
	if len(tfm.ExperimentIds) > 0 {
		ws.Include("experiment_id in (?)", bun.In(tfm.ExperimentIds))
	}
	q = ws.Apply(q)

	var exist []int64
	err := q.Scan(ctx, &exist)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		fmt.Printf("error: %v\n", err)
		return nil, err
	}

	// step 2: figure out what was missing and what has appeared
	missing, appeared, err := processKnown(known, exist)
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

	// step 4: emit deletions and udpates to the client
	out = append(out, TrialsDeletedMsg{missing}.PreparedMessage())
	for _, msg := range trialMsgs {
		out = append(out, msg.PreparedMessage())
	}
	return out, nil
}

// When a user submits a new TrialFilterMod, we scrape the database for initial matches.
func (tfm TrialFilterMod) Modify(ctx context.Context) (
	[]*websocket.PreparedMessage, error,
) {
	if len(tfm.TrialIds) == 0 && len(tfm.ExperimentIds) == 0 {
		return nil, nil
	}
	var trialMsgs []*TrialMsg
	q := db.Bun().NewSelect().Model(&trialMsgs)

	// Use WhereSince to build a complex WHERE clause.
	ws := WhereSince{Since: tfm.Since}
	if len(tfm.TrialIds) > 0 {
		ws.Include("id in (?)", bun.In(tfm.TrialIds))
	}
	if len(tfm.ExperimentIds) > 0 {
		ws.Include("experiment_id in (?)", bun.In(tfm.ExperimentIds))
	}
	q = ws.Apply(q)

	err := q.Scan(ctx)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		fmt.Printf("error: %v\n", err)
		return nil, err
	}

	var out []*websocket.PreparedMessage
	for _, msg := range trialMsgs {
		out = append(out, msg.PreparedMessage())
	}
	return out, nil
}

type TrialFilterMaker struct {
	TrialIds      map[int]bool
	ExperimentIds map[int]bool
}

func NewTrialFilterMaker() FilterMaker[*TrialMsg] {
	return &TrialFilterMaker{make(map[int]bool), make(map[int]bool)}
}

func (ts *TrialFilterMaker) AddSpec(spec FilterMod) {
	tSpec := spec.(TrialFilterMod)
	for _, id := range tSpec.TrialIds {
		ts.TrialIds[id] = true
	}
	for _, id := range tSpec.ExperimentIds {
		ts.ExperimentIds[id] = true
	}
}

func (ts *TrialFilterMaker) DropSpec(spec FilterMod) {
	tSpec := spec.(TrialFilterMod)
	for _, id := range tSpec.TrialIds {
		delete(ts.TrialIds, id)
	}
	for _, id := range tSpec.ExperimentIds {
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
	for id, _ := range ts.TrialIds {
		trialIds[id] = true
	}
	for id, _ := range ts.ExperimentIds {
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
