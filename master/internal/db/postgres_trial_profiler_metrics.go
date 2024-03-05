package db

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/determined-ai/determined/master/pkg/protoutils"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/pkg/errors"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/trialv1"
)

// InsertTrialProfilerMetricsBatch inserts a batch of metrics into the database.
func (db *PgDB) InsertTrialProfilerMetricsBatch(
	values []float32, batches []int32, timestamps []time.Time, labels []byte,
) error {
	_, err := db.sql.Exec(`
INSERT INTO trial_profiler_metrics (values, batches, ts, labels)
VALUES ($1, $2, $3, $4)
`, values, batches, timestamps, labels)
	return err
}

const (
	cpu     string = "cpu"
	gpu     string = "gpu"
	disk    string = "disk"
	network string = "network"
	memory  string = "memory"
)

func groupFromLabelName(n string) string {
	switch {
	case strings.HasPrefix(n, "cpu_"):
		return cpu
	case strings.HasPrefix(n, "gpu_"):
		return gpu
	case strings.HasPrefix(n, "disk_"):
		return disk
	case strings.HasPrefix(n, "net_"):
		return network
	case strings.HasPrefix(n, "mem_"):
		return memory
	default:
		return ""
	}
}

// GetTrialProfilerMetricsBatches gets a batch of profiler metric batches from the database.
// This method is for backwards compatibility and should be deprecated in the future in favor of
// generics metrics APIs.
func (db *PgDB) GetTrialProfilerMetricsBatches(
	labels *trialv1.TrialProfilerMetricLabels, offset, limit int,
) (model.TrialProfilerMetricsBatchBatch, error) {
	metricGroup := groupFromLabelName(labels.Name)

	// Translate old schema to new, which formats metrics with label IDs as keys.
	jsonPath := "metrics"
	if labels.AgentId != "" {
		if labels.GpuUuid != "" {
			jsonPath = fmt.Sprintf("metrics->'%s'->'%s'", labels.AgentId, labels.GpuUuid)
		} else {
			jsonPath = fmt.Sprintf("metrics->'%s'", labels.AgentId)
		}
	}

	rows, err := db.sql.Queryx(`
SELECT
    `+jsonPath+`->>$1 AS values,
    end_time AS timestamps
FROM metrics
WHERE partition_type='PROFILING'
AND trial_id=$2
AND metric_group=$3
AND `+jsonPath+` ? $1 
ORDER by id
OFFSET $4 LIMIT $5`, labels.Name, labels.TrialId, metricGroup, offset, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pBatches []*trialv1.TrialProfilerMetricsBatch
	for rows.Next() {
		var resValue *float32
		var resTime *time.Time
		if err := rows.Scan(&resValue, &resTime); err != nil {
			return nil, errors.Wrap(err, "querying profiler metric batch")
		}

		pBatch := &trialv1.TrialProfilerMetricsBatch{
			Values:     []float32{*resValue},
			Timestamps: []*timestamppb.Timestamp{protoutils.ToTimestamp(*resTime)},
			Labels:     labels,
		}
		if err != nil {
			return nil, errors.Wrap(err, "converting batch to protobuf")
		}

		pBatches = append(pBatches, pBatch)
	}
	return pBatches, nil
}

// GetTrialProfilerAvailableSeries returns all available system profiling metric names.
// This method is to be deprecated in the future in place of generic metrics APIs.
func (db *PgDB) GetTrialProfilerAvailableSeries(
	ctx context.Context, trialID int32,
) ([]*trialv1.TrialProfilerMetricLabels, error) {
	out := []struct {
		Group   string
		Metrics map[string]interface{}
	}{}

	// Since all system metrics are reflected in each report, get the last row reported for each group.
	query := Bun().NewSelect().Table("metrics").
		DistinctOn("metric_group, _agent_id").
		ColumnExpr("metric_group as group").
		ColumnExpr("jsonb_object_keys(metrics) as _agent_id").
		Column("metrics").
		Where("partition_type = 'PROFILING'").
		Where("trial_id = ?", trialID).
		Order("metric_group").
		Order("_agent_id").
		Order("end_time DESC")

	err := query.Scan(ctx, &out)
	if err != nil {
		return nil, err
	}
	var seriesLabels []*trialv1.TrialProfilerMetricLabels
	for _, m := range out {
		for aID, aM := range m.Metrics {
			// Top-level keys are always Agent IDs.
			agentID := aID
			if metrics, ok := aM.(map[string]interface{}); ok {
				for k, v := range metrics {
					// Certain metric groups have additional labels.
					if labeledMetrics, ok := v.(map[string]interface{}); ok {
						for name := range labeledMetrics {
							label := trialv1.TrialProfilerMetricLabels{
								TrialId:    trialID,
								AgentId:    agentID,
								MetricType: trialv1.TrialProfilerMetricLabels_PROFILER_METRIC_TYPE_SYSTEM,
							}
							switch m.Group {
							case gpu:
								label.GpuUuid = k
							case disk:
								// Skip these for now because the Web UI needs to be aware.
								// label.DiskPath = k
								continue
							}
							label.Name = name
							seriesLabels = append(seriesLabels, &label)
						}
					} else {
						label := trialv1.TrialProfilerMetricLabels{
							TrialId:    trialID,
							AgentId:    agentID,
							MetricType: trialv1.TrialProfilerMetricLabels_PROFILER_METRIC_TYPE_SYSTEM,
						}
						label.Name = k
						seriesLabels = append(seriesLabels, &label)
					}
				}
			}
		}
	}

	return seriesLabels, nil
}
