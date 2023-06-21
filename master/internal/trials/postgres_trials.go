package trials

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/protoutils"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

const (
	batches = "batches"
)

// TrialsAugmented shows provides information about a Trial.
type TrialsAugmented struct {
	bun.BaseModel         `bun:"table:trials_augmented_view,alias:trials_augmented_view"`
	TrialID               int32              `bun:"trial_id"`
	State                 string             `bun:"state"`
	Hparams               model.JSONObj      `bun:"hparams"`
	TrainingMetrics       map[string]float64 `bun:"training_metrics,json_use_number"`
	ValidationMetrics     map[string]float64 `bun:"validation_metrics,json_use_number"`
	Tags                  map[string]string  `bun:"tags"`
	StartTime             time.Time          `bun:"start_time"`
	EndTime               time.Time          `bun:"end_time"`
	SearcherType          string             `bun:"searcher_type"`
	ExperimentID          int32              `bun:"experiment_id"`
	ExperimentName        string             `bun:"experiment_name"`
	ExperimentDescription string             `bun:"experiment_description"`
	ExperimentLabels      []string           `bun:"experiment_labels"`
	UserID                int32              `bun:"user_id"`
	ProjectID             int32              `bun:"project_id"`
	WorkspaceID           int32              `bun:"workspace_id"`
	TotalBatches          int32              `bun:"total_batches"`
	SearcherMetric        string             `bun:"searcher_metric"`
	SearcherMetricValue   float64            `bun:"searcher_metric_value"`
	SearcherMetricLoss    float64            `bun:"searcher_metric_loss"`

	RankWithinExp int32 `bun:"rank,scanonly"`
}

// Proto converts an Augmented Trial to its protobuf representation.
func (t *TrialsAugmented) Proto() *apiv1.AugmentedTrial {
	return &apiv1.AugmentedTrial{
		TrialId:               t.TrialID,
		State:                 trialv1.State(trialv1.State_value["STATE_"+t.State]),
		Hparams:               protoutils.ToStruct(t.Hparams),
		TrainingMetrics:       protoutils.ToStruct(t.TrainingMetrics),
		ValidationMetrics:     protoutils.ToStruct(t.ValidationMetrics),
		Tags:                  protoutils.ToStruct(t.Tags),
		StartTime:             protoutils.ToTimestamp(t.StartTime),
		EndTime:               protoutils.ToTimestamp(t.EndTime),
		SearcherType:          t.SearcherType,
		ExperimentId:          t.ExperimentID,
		ExperimentName:        t.ExperimentName,
		ExperimentDescription: t.ExperimentDescription,
		ExperimentLabels:      t.ExperimentLabels,
		UserId:                t.UserID,
		ProjectId:             t.ProjectID,
		WorkspaceId:           t.WorkspaceID,
		TotalBatches:          t.TotalBatches,
		RankWithinExp:         t.RankWithinExp,
		SearcherMetric:        t.SearcherMetric,
		SearcherMetricValue:   t.SearcherMetricValue,
		SearcherMetricLoss:    t.SearcherMetricLoss,
	}
}

// TrialsCollection is a collection of Trials matching a set of TrialFilters.
type TrialsCollection struct {
	ID        int32               `bun:"id,pk,autoincrement"`
	UserID    int32               `bun:"user_id"`
	ProjectID int32               `bun:"project_id"`
	Name      string              `bun:"name"`
	Filters   *apiv1.TrialFilters `bun:"filters,type:jsonb"`
	Sorter    *apiv1.TrialSorter  `bun:"sorter,type:jsonb"`
}

// Proto converts TrialsCollection to proto representation.
func (tc *TrialsCollection) Proto() *apiv1.TrialsCollection {
	return &apiv1.TrialsCollection{
		Id:        tc.ID,
		UserId:    tc.UserID,
		ProjectId: tc.ProjectID,
		Name:      tc.Name,
		Filters:   tc.Filters,
		Sorter:    tc.Sorter,
	}
}

// QueryTrialsOrderMap is a map of OrderBy choices to Sort Choices.
var QueryTrialsOrderMap = map[apiv1.OrderBy]db.SortDirection{
	apiv1.OrderBy_ORDER_BY_UNSPECIFIED: db.SortDirectionAsc,
	apiv1.OrderBy_ORDER_BY_ASC:         db.SortDirectionAsc,
	apiv1.OrderBy_ORDER_BY_DESC:        db.SortDirectionDescNullsLast,
}

// This allows dot on top of whats allowed in existing regex validField.
var safeString = regexp.MustCompile(`^[a-zA-Z0-9_\.\-]+$`)

func hParamAccessor(hp string) string {
	pathElementsRaw := strings.Split(hp, ".")
	pathElements := []string{"hparams"}
	for _, n := range pathElementsRaw {
		pathElements = append(pathElements, fmt.Sprintf("'%s'", n))
	}
	path := strings.Join(pathElements[:len(pathElements)-1], "->")
	key := pathElements[len(pathElements)-1]

	return fmt.Sprintf("(%s->>%s)::float8", path, key)
}

// BuildTrialPatchQuery creates an UpdateQuery according to the provided patch.
func BuildTrialPatchQuery(payload *apiv1.TrialPatch) (*bun.UpdateQuery, error) {
	q := db.Bun().NewUpdate().Table("trials")

	if len(payload.AddTag) > 0 || len(payload.RemoveTag) > 0 {
		addTags := map[string]string{}
		for _, tag := range payload.AddTag {
			addTags[tag.Key] = ""
		}

		removeTags := []string{}
		for _, tag := range payload.RemoveTag {
			removeTags = append(removeTags, tag.Key)
		}

		q = q.Set("tags = (tags || ?) - ?::text[]", addTags, pgdialect.Array(removeTags))
	}
	return q, nil
}

// TrialsColumnForNamespace returns the correct namespace for a TrialSorter.
func TrialsColumnForNamespace(namespace apiv1.TrialSorter_Namespace,
	field string,
) (string, error) {
	if !safeString.MatchString(field) {
		return "", fmt.Errorf("%s filter %s contains possible SQL injection", namespace, field)
	}
	switch namespace {
	case apiv1.TrialSorter_NAMESPACE_UNSPECIFIED:
		return field, nil
	case apiv1.TrialSorter_NAMESPACE_HPARAMS:
		return hParamAccessor(field), nil
	case apiv1.TrialSorter_NAMESPACE_TRAINING_METRICS:
		return fmt.Sprintf("(training_metrics->>'%s')::float8", field), nil
	case apiv1.TrialSorter_NAMESPACE_VALIDATION_METRICS:
		return fmt.Sprintf("(validation_metrics->>'%s')::float8", field), nil
	default:
		return field, nil
	}
}

// BuildFilterTrialsQuery queries for Trials matching the supplied TrialFilters.
func BuildFilterTrialsQuery(filters *apiv1.TrialFilters, selectAll bool) (*bun.SelectQuery, error) {
	// FilterTrials filters trials according to filters

	q := db.Bun().NewSelect().Model((*TrialsAugmented)(nil))

	rankFilterApplied := filters.RankWithinExp != nil && filters.RankWithinExp.Rank != 0

	if rankFilterApplied || selectAll {
		r := filters.RankWithinExp

		if r == nil {
			r = &apiv1.TrialFilters_RankWithinExp{
				Rank: 0,
				Sorter: &apiv1.TrialSorter{
					Namespace: apiv1.TrialSorter_NAMESPACE_UNSPECIFIED,
					Field:     "trial_id",
					OrderBy:   apiv1.OrderBy_ORDER_BY_ASC,
				},
			}
		}

		columnExpr, err := TrialsColumnForNamespace(r.Sorter.Namespace, r.Sorter.Field)
		if err != nil {
			return nil, fmt.Errorf("possible unsafe filters, %f", err)
		}
		rankExpr := fmt.Sprintf(
			`ROW_NUMBER() OVER(PARTITION BY experiment_id ORDER BY %s  %s) as rank`,
			columnExpr,
			QueryTrialsOrderMap[r.Sorter.OrderBy])

		rankQ := db.Bun().NewSelect().
			Model((*TrialsAugmented)(nil)).
			ColumnExpr("trial_id as t_id").
			ColumnExpr(rankExpr)

		q.With("ranking", rankQ).
			Join("join ranking on ranking.t_id = trials_augmented_view.trial_id")

		if rankFilterApplied {
			q.Where("ranking.rank <= ?", r.Rank)
		}
		if selectAll {
			q.ColumnExpr("trials_augmented_view.*, ranking.rank")
		}
	}

	if len(filters.Tags) > 0 {
		tagKeys := []string{}
		for _, tag := range filters.Tags {
			tagKeys = append(tagKeys, tag.Key)
		}
		// bun please ignore the first question mark,
		// it is an operator, not a placeholder
		q.Where("tags ?| ?", bun.Safe("?"), pgdialect.Array(tagKeys))
	}

	if len(filters.TrialIds) > 0 {
		q.Where("trial_id IN (?)", bun.In(filters.TrialIds))
	}

	if len(filters.ExperimentIds) > 0 {
		q.Where("experiment_id IN (?)", bun.In(filters.ExperimentIds))
	}
	if len(filters.ProjectIds) > 0 {
		q.Where("project_id IN (?)", bun.In(filters.ProjectIds))
	}
	if len(filters.WorkspaceIds) > 0 {
		q.Where("workspace_id IN (?)", bun.In(filters.WorkspaceIds))
	}

	for _, f := range filters.ValidationMetrics {
		if !safeString.MatchString(f.Name) {
			return nil, fmt.Errorf("metric filter %s contains possible SQL injection", f.Name)
		}
		_, err := db.ApplyDoubleFieldFilter(
			q,
			fmt.Sprintf("(validation_metrics->>'%s')::float8", f.Name),
			f.Filter,
		)
		if err != nil {
			return nil, err
		}
	}

	for _, f := range filters.TrainingMetrics {
		if !safeString.MatchString(f.Name) {
			return nil, fmt.Errorf("metric filter %s contains possible SQL injection", f.Name)
		}
		_, err := db.ApplyDoubleFieldFilter(
			q,
			fmt.Sprintf("(training_metrics->>'%s')::float8", f.Name),
			f.Filter,
		)
		if err != nil {
			return nil, err
		}
	}

	for _, f := range filters.Hparams {
		_, err := db.ApplyDoubleFieldFilter(
			q,
			fmt.Sprintf("(%s)::float8", hParamAccessor(f.Name)),
			f.Filter,
		)
		if err != nil {
			return nil, err
		}
	}

	if filters.Searcher != "" {
		q.Where("searcher_type = ?", filters.Searcher)
	}
	if len(filters.UserIds) > 0 {
		q.Where("user_id IN (?)", bun.In(filters.UserIds))
	}

	if filters.StartTime != nil {
		_, err := db.ApplyTimestampFieldFilter(q, bun.Ident("start_time"), filters.StartTime)
		if err != nil {
			return nil, err
		}
	}

	if filters.EndTime != nil {
		_, err := db.ApplyTimestampFieldFilter(q, bun.Ident("end_time"), filters.EndTime)
		if err != nil {
			return nil, err
		}
	}

	if len(filters.States) > 0 {
		states := []string{}
		for _, s := range filters.States {
			states = append(states, strings.TrimPrefix(s.String(), "STATE_"))
		}
		q.Where("state in (?)", bun.In(states))
	}

	if filters.SearcherMetric != "" {
		q.Where("searcher_metric = ?", filters.SearcherMetric)
	}

	if filters.SearcherMetricValue != nil {
		_, err := db.ApplyDoubleFieldFilter(
			q,
			bun.Ident("searcher_metric_value"),
			filters.SearcherMetricValue,
		)
		if err != nil {
			return nil, err
		}
	}

	return q, nil
}

// MetricsTimeSeries returns a time-series of the specified metric in the specified
// trial.
func MetricsTimeSeries(trialID int32, startTime time.Time,
	metricNames []string,
	startBatches int, endBatches int, xAxisMetricLabels []string,
	maxDatapoints int, timeSeriesColumn string,
	timeSeriesFilter *commonv1.PolymorphicFilter, metricType model.MetricType) (
	metricMeasurements []db.MetricMeasurements, err error,
) {
	var queryColumn, orderColumn string
	metricsObjectName := model.TrialMetricsJSONPath(
		metricType == model.ValidationMetricType)
	// The data for batches and column are stored under different column names
	switch timeSeriesColumn {
	case "batches":
		queryColumn = "total_batches"
	case "time":
		queryColumn = "end_time"
	default:
		queryColumn = timeSeriesColumn
	}
	subq := db.BunSelectMetricsQuery(metricType, false).Table("metrics").
		ColumnExpr("(select setseed(1)) as _seed").
		ColumnExpr("total_batches as batches").
		ColumnExpr("trial_id").ColumnExpr("end_time as time")

	type summary struct {
		bun.BaseModel `bun:"table:trials"`
		Metrics       map[string]any
	}
	var summaryMetrics summary
	if err := db.Bun().NewSelect().Table("trials").
		ColumnExpr("summary_metrics->? AS metrics", metricsObjectName).
		Where("id = ?", trialID).
		Scan(context.TODO(), &summaryMetrics); err != nil {
		return nil, fmt.Errorf("getting summary metrics for trial %d: %w", trialID, err)
	}

	for _, metricName := range append(metricNames, "epoch") {
		metricType := db.MetricTypeString
		if curSummary, ok := summaryMetrics.Metrics[metricName].(map[string]any); ok {
			if m, ok := curSummary["type"].(string); ok {
				metricType = m
			}
		}

		cast := "text"
		switch metricType {
		case db.MetricTypeNumber:
			cast = "float8"
		case db.MetricTypeBool:
			cast = "boolean"
		}
		subq = subq.ColumnExpr("(metrics->?->>?)::? as ?",
			metricsObjectName, metricName, bun.Safe(cast), bun.Ident(metricName))
	}

	subq = subq.Where("trial_id = ?", trialID).OrderExpr("random()").
		Limit(maxDatapoints)
	switch timeSeriesFilter {
	case nil:
		orderColumn = batches
		subq = subq.Where("total_batches >= ?", startBatches).
			Where("total_batches <= 0 OR total_batches <= ?", endBatches).
			Where("end_time > ?", startTime)
	default:
		orderColumn = timeSeriesColumn
		subq, err = db.ApplyPolymorphicFilter(subq, queryColumn, timeSeriesFilter)
		if err != nil {
			return metricMeasurements, errors.Wrapf(err, "failed to get metrics to sample for experiment")
		}
	}

	metricMeasurements = []db.MetricMeasurements{}
	var results []map[string]interface{}
	err = db.Bun().NewSelect().TableExpr("(?) as downsample", subq).
		OrderExpr(orderColumn).Scan(context.TODO(), &results)
	if err != nil {
		return metricMeasurements, errors.Wrapf(err, "failed to get metrics to sample for experiment")
	}

	selectMetrics := map[string]bool{}

	for i := range metricNames {
		selectMetrics[metricNames[i]] = true
	}

	for i := range results {
		valuesMap := make(map[string]interface{})
		for mName, mVal := range results[i] {
			if selectMetrics[mName] {
				valuesMap[mName] = mVal
			}
		}
		epoch := new(int32)
		if results[i]["epoch"] != nil {
			if e, ok := results[i]["epoch"].(float64); ok {
				*epoch = int32(e)
			} else {
				return nil, fmt.Errorf(
					"metric 'epoch' has nonnumeric value reported value='%v'", results[i]["epoch"])
			}
		}
		var endTime time.Time
		if results[i]["time"] == nil {
			endTime = time.Time{}
		} else {
			endTime = results[i]["time"].(time.Time)
		}
		metricM := db.MetricMeasurements{
			Batches: uint(results[i]["batches"].(int64)),
			Time:    endTime,
			Epoch:   epoch,
			TrialID: int32(results[i]["trial_id"].(int64)),
			Values:  valuesMap,
		}

		metricMeasurements = append(metricMeasurements, metricM)
	}
	return metricMeasurements, nil
}
