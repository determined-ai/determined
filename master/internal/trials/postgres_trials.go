package trials

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
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
