package stream

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/stream"
)

const (
	// MetricsUpsertKey specifies the key for upsert metrics.
	MetricsUpsertKey = "metric"
)

// MetricMsg is a stream.Msg.
// determined:streamable
type MetricMsg struct {
	bun.BaseModel `bun:"table:metrics"`

	// immutable attributes
	TrialID       int                    `bun:"trial_id" json:"trial_id"`
	TrialRunID    int                    `bun:"trial_run_id" json:"trial_run_id"`
	EndTime       *time.Time             `bun:"end_time" json:"end_time"`
	Metrics       JSONB                  `bun:"metrics" json:"metrics"`
	TotalBatches  int                    `bun:"total_batches" json:"total_batches"`
	MetricGroup   string                 `bun:"metric_group" json:"metric_group"`
	PartitionType db.MetricPartitionType `bun:"partition_type" json:"partition_type"`

	// mutable attributes
	Archived bool `bun:"archived" json:"archived"`

	// metadata
	Seq int64 `bun:"seq" json:"seq"`

	// permission scope
	WorkspaceID int `json:"workspace_id,omitempty"`
}

// SeqNum gets the SeqNum from a MetricMsg.
func (mm *MetricMsg) SeqNum() int64 {
	return mm.Seq
}

// UpsertMsg creates a Metric stream upsert message.
func (mm *MetricMsg) UpsertMsg() stream.UpsertMsg {
	// omit workspaceID since it wasn't part of the original row
	mm.WorkspaceID = 0
	return stream.UpsertMsg{
		JSONKey: MetricsUpsertKey,
		Msg:     mm,
	}
}

// DeleteMsg creates a Metric stream delete message.
func (mm *MetricMsg) DeleteMsg() stream.DeleteMsg {
	panic("streaming metric is append-only, delete messages are not supported")
}

// MetricSubscriptionSpec is what a user submits to define a Metric subscription.
// determined:streamable
type MetricSubscriptionSpec struct {
	TrialIds []int `json:"trial_ids"`
	Since    int64 `json:"since"`
}

func getMetricMsgsWithWorkspaceID(metricMsgs []*MetricMsg) *bun.SelectQuery {
	q := db.Bun().NewSelect().Model(&metricMsgs).
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
	_ string, // known is not needed since metrics are append-only
	spec MetricSubscriptionSpec,
) (
	[]stream.PreparableMessage, error,
) {
	var out []stream.PreparableMessage

	if len(spec.TrialIds) == 0 {
		// empty subscription: noop
		return nil, nil
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
	if len(spec.TrialIds) > 0 {
		ws.Include("trial_id in (?)", bun.In(spec.TrialIds))
	}
	q = ws.Apply(q)

	var exist []int64
	err = q.Scan(ctx, &exist)
	if err != nil && errors.Cause(err) != sql.ErrNoRows {
		log.Errorf("error: %v\n", err)
		return nil, err
	}

	// step 2: skipped, since metrics are append-only
	// step 3: hydrate appeared IDs into full MetricMsgs
	var metricMsgs []*MetricMsg
	if len(exist) > 0 {
		query := getMetricMsgsWithWorkspaceID(metricMsgs).
			Where("metric_msg.id in (?)", bun.In(exist))
		query = permFilter(query)
		err := query.Scan(ctx, &metricMsgs)
		if err != nil && errors.Cause(err) != sql.ErrNoRows {
			log.Errorf("error: %v\n", err)
			return nil, err
		}
	}

	// step 4: emit updates to the client
	for _, msg := range metricMsgs {
		out = append(out, msg.UpsertMsg())
	}
	return out, nil
}

// MetricMakeFilter creates a MetricMsg filter based on the given MetricSubscriptionSpec.
func MetricMakeFilter(spec *MetricSubscriptionSpec) (func(*MetricMsg) bool, error) {
	// Should this filter even run?
	if len(spec.TrialIds) == 0 {
		return nil, fmt.Errorf("invalid subscription spec arguments: %v", spec.TrialIds)
	}

	logrus.Debug("Spec.TrialIDs: %d", spec.TrialIds)
	// Make a copy of the map, because the filter must run safely off-thread.
	trialIds := make(map[int]struct{})
	for _, id := range spec.TrialIds {
		if id <= 0 {
			return nil, fmt.Errorf("invalid trial id: %d", id)
		}
		trialIds[id] = struct{}{}
	}
	logrus.Debug("TrialIDs:%v", trialIds)

	// return a closure around our copied map
	return func(msg *MetricMsg) bool {
		logrus.Errorf("MetricMsg.TrialID: %d", msg.TrialID)
		logrus.Errorf("trialIds: %v", trialIds)
		_, ok := trialIds[msg.TrialID]
		logrus.Errorf("Did it pass?: %t", ok)
		return ok
	}, nil
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
