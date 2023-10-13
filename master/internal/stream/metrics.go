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

// MetricsDeleteKey specifies the key for delete Metrics.
const MetricsDeleteKey = "metrics_deleted"

// MetricMsg is a stream.Msg.
// determined:streamable
type MetricMsg struct {
	bun.BaseModel `bun:"table:metrics"`

	// immutable attributes
	ID            int                    `bun:"id,pk" json:"id"`
	TrialID       int                    `bun:"trial_id" json:"trial_id"`
	TrialRunID    int                    `bun:"trial_run_id" json:"trial_run_id"`
	EndTime       *time.Time             `bun:"end_time" json:"end_time"`
	Metrics       JSONB                  `bun:"metrics" json:"metrics"`
	TotalBatches  int                    `bun:"total_batches" json:"total_batches"`
	MetricGroup   string                 `bun:"metric_group" json:"metric_group"`
	PartitionType db.MetricPartitionType `bun:"partition_type" json:"partition_type"`
	Archived      bool                   `bun:"archived" json:"archived"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`

	// permission scope
	WorkspaceID int `json:"-"`

	upsertCache *websocket.PreparedMessage
	deleteCache *websocket.PreparedMessage
}

// SeqNum gets the SeqNum from a MetricMsg.
func (mm *MetricMsg) SeqNum() int64 {
	return mm.Seq
}

// UpsertMsg creates a Metric upserted prepared message.
func (mm *MetricMsg) UpsertMsg() *websocket.PreparedMessage {
	wrapper := struct {
		Metric *MetricMsg `json:"metric"`
	}{mm}
	return prepareMessageWithCache(wrapper, &mm.upsertCache)
}

// DeleteMsg creates a Trial deleted prepared message.
func (mm *MetricMsg) DeleteMsg() *websocket.PreparedMessage {
	deleted := strconv.FormatInt(int64(mm.ID), 10)
	return newDeletedMsgWithCache(MetricsDeleteKey, deleted, &mm.deleteCache)
}

// MetricSubscriptionSpec is what a user submits to define a Metric subscription.
// determined:streamable
type MetricSubscriptionSpec struct {
	MetricIds []int `json:"metric_ids"`
	Since     int64 `json:"since"`
}

func getMetricMsgsWithWorkspaceID(metricMsgs []*MetricMsg) *bun.SelectQuery {
	q := db.Bun().NewSelect().Model(&metricMsgs).
		Column("id").
		Column("trial_id").
		Column("trial_run_id").
		Column("end_time").
		Column("metrics").
		Column("total_batches").
		Column("metric_group").
		Column("partition_type").
		Column("archived").
		Column("seq").
		Column("projects.workspace_id").
		Join("JOIN trials ON metric_msg.trial_id = trials.id").
		Join("JOIN experiments ON trials.experiment_id = experiments.id").
		Join("JOIN projects ON experiments.project_id = projects.id")
	return q
}

// MetricCollectStartupMsgs collects MetricMsg's that were missed prior to startup.
func MetricCollectStartupMsgs(
	ctx context.Context,
	user model.User,
	known string,
	spec MetricSubscriptionSpec,
) (
	[]*websocket.PreparedMessage, error,
) {
	var out []*websocket.PreparedMessage

	if len(spec.MetricIds) == 0 {
		// empty subscription: everything known should be returned as deleted
		out = append(out, newDeletedMsg(MetricsDeleteKey, known))
		return out, nil
	}
	// step 0: get user's permitted access scopes
	accessMap, err := AuthZProvider.Get().GetMetricStreamableScopes(ctx, user)
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
		Table("metrics").
		Column("metrics.id").
		Join("JOIN trials t ON metrics.trial_id = t.id").
		Join("JOIN experiments e ON t.experiment_id = e.id").
		Join("JOIN projects p ON e.project_id = p.id").
		OrderExpr("metrics.id ASC")
	q = permFilter(q)

	// Ignore mmf.Since, because we want appearances, which might not be have seq > spec.Since.
	ws := stream.WhereSince{Since: 0}
	if len(spec.MetricIds) > 0 {
		ws.Include("metrics.id in (?)", bun.In(spec.MetricIds))
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

	// step 3: hydrate appeared IDs into full MetricMsgs
	var metricMsgs []*MetricMsg
	if len(appeared) > 0 {
		query := getMetricMsgsWithWorkspaceID(metricMsgs).
			Where("metric_msg.id in (?)", bun.In(appeared))
		query = permFilter(query)
		err := query.Scan(ctx, &metricMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 4: emit deletions and updates to the client
	if len(missing) > 0 {
		out = append(out, newDeletedMsg(MetricsDeleteKey, missing))
	}
	for _, msg := range metricMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// MetricCollectSubscriptionModMsgs scrapes the database when a
// user submits a new MetricSubscriptionSpec for initial matches.
func MetricCollectSubscriptionModMsgs(ctx context.Context, addSpec MetricSubscriptionSpec) (
	[]*websocket.PreparedMessage, error,
) {
	if len(addSpec.MetricIds) == 0 {
		return nil, nil
	}
	var metricMsgs []*MetricMsg
	q := getMetricMsgsWithWorkspaceID(metricMsgs)

	// Use WhereSince to build a complex WHERE clause.
	ws := stream.WhereSince{Since: addSpec.Since}
	if len(addSpec.MetricIds) > 0 {
		ws.Include("id in (?)", bun.In(addSpec.MetricIds))
	}
	q = ws.Apply(q)

	err := q.Scan(ctx)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("error: %v\n", err)
		return nil, err
	}

	var out []*websocket.PreparedMessage
	for _, msg := range metricMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// MetricFilterMaker tracks the metric id's that are to be filtered for.
type MetricFilterMaker struct {
	MetricIds map[int]bool
}

// NewMetricFilterMaker creates a new FilterMaker.
func NewMetricFilterMaker() FilterMaker[*MetricMsg, MetricSubscriptionSpec] {
	return &MetricFilterMaker{make(map[int]bool)}
}

// AddSpec adds MetricIds specified in MetricSubscriptionSpec.
func (ms *MetricFilterMaker) AddSpec(spec MetricSubscriptionSpec) {
	for _, id := range spec.MetricIds {
		ms.MetricIds[id] = true
	}
}

// DropSpec removes MetricIds specified in MetricSubscriptionSpec.
func (ms *MetricFilterMaker) DropSpec(spec MetricSubscriptionSpec) {
	for _, id := range spec.MetricIds {
		delete(ms.MetricIds, id)
	}
}

// MakeFilter returns a function that determines if a MetricMsg based on
// the MetricFilterMaker's spec.
func (ms *MetricFilterMaker) MakeFilter() func(*MetricMsg) bool {
	// Should this filter even run?
	if len(ms.MetricIds) == 0 {
		return nil
	}

	// Make a copy of the map, because the filter must run safely off-thread.
	metricIds := make(map[int]bool)
	for id := range ms.MetricIds {
		metricIds[id] = true
	}

	// return a closure around our copied map
	return func(msg *MetricMsg) bool {
		if _, ok := metricIds[msg.ID]; ok {
			return true
		}
		return false
	}
}

// MetricMakePermissionFilter returns a function that checks if a MetricMsg
// is in scope of the user permissions.
func MetricMakePermissionFilter(
	ctx context.Context,
	user model.User,
) (func(*MetricMsg) bool, error) {
	accessScopeSet, err := AuthZProvider.Get().GetMetricStreamableScopes(
		ctx,
		user,
	)
	if err != nil {
		return nil, err
	}

	switch {
	case accessScopeSet[model.GlobalAccessScopeID]:
		// user has global access for viewing Metrics
		return func(msg *MetricMsg) bool { return true }, nil
	default:
		return func(msg *MetricMsg) bool {
			return accessScopeSet[model.AccessScopeID(msg.WorkspaceID)]
		}, nil
	}
}
