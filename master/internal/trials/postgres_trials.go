package trials

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

const (
	batches = "batches"
)

// MetricsTimeSeries returns a time-series of the specified metric in the specified
// trial.
func MetricsTimeSeries(trialID int32, startTime time.Time,
	metricNames []string,
	startBatches int, endBatches int, xAxisMetricLabels []string,
	maxDatapoints int, timeSeriesColumn string,
	timeSeriesFilter *commonv1.PolymorphicFilter, metricGroup model.MetricGroup) (
	metricMeasurements []db.MetricMeasurements, err error,
) {
	var queryColumn, orderColumn string
	metricsObjectName := model.TrialMetricsJSONPath(
		metricGroup == model.ValidationMetricGroup)
	// The data for batches and column are stored under different column names
	switch timeSeriesColumn {
	case "batches":
		queryColumn = "total_batches"
	case "time":
		queryColumn = "end_time"
	default:
		queryColumn = strings.ReplaceAll(timeSeriesColumn, ".", "·")
	}
	subq := db.BunSelectMetricsQuery(metricGroup, false).Table("metrics").
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
		metricGroup := db.MetricTypeString
		if curSummary, ok := summaryMetrics.Metrics[metricName].(map[string]any); ok {
			if m, ok := curSummary["type"].(string); ok {
				metricGroup = m
			}
		}

		cast := "text"
		switch metricGroup {
		case db.MetricTypeNumber:
			cast = "float8"
		case db.MetricTypeBool:
			cast = "boolean"
		}
		subq = subq.ColumnExpr("(metrics->?->>?)::? as ?", metricsObjectName,
			metricName, bun.Safe(cast), bun.Ident(strings.ReplaceAll(metricName, ".", "·")))
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
		orderColumn = strings.ReplaceAll(timeSeriesColumn, ".", "·")
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

	selectMetrics := map[string]string{}

	for i := range metricNames {
		selectMetrics[strings.ReplaceAll(metricNames[i], ".", "·")] = metricNames[i]
	}

	for i := range results {
		valuesMap := make(map[string]interface{})
		for mName, mVal := range results[i] {
			if selectMetrics[mName] != "" {
				valuesMap[selectMetrics[mName]] = mVal
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

// CreateTrialSourceInfo creates a TrialSourceInfo object, which allows us to keep
// track of the linkage between an inference/fine tuning trial and its checkpoint/model version.
func CreateTrialSourceInfo(ctx context.Context, tsi *trialv1.TrialSourceInfo,
) (*apiv1.ReportTrialSourceInfoResponse, error) {
	resp := &apiv1.ReportTrialSourceInfoResponse{}
	query := db.Bun().NewInsert().Model(tsi).
		Value("trial_source_info_type", "?", tsi.TrialSourceInfoType.String()).
		Returning("trial_id").Returning("checkpoint_uuid")
	if tsi.ModelId == nil {
		query.ExcludeColumn("model_id")
	}
	if tsi.ModelVersion == nil {
		query.ExcludeColumn("model_version")
	}
	_, err := query.Exec(ctx, resp)
	return resp, err
}
