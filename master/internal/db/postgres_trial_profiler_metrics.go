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
	case strings.HasPrefix(n, "memory_"):
		return memory
	default:
		return ""
	}
}

// GetTrialProfilerMetricsBatches gets a batch of profiler metric batches from the database.
// This method is for backwards compatibility and should be deprecated in the future in favor of
// generics metrics APIs.
//
// Profiler metrics are stored in the metrics table as a nested JSON mapping of labels to values.
// All profiler metrics are associated with an agent ID, but certain metrics (i.e. gpu_util) may
// be associated with other labels. For example:
//
//	{
//		"agent-ID-1": {
//			"GPU-UUID-1": {
//				"gpu_util": 0.12,
//				"gpu_free_memory": 0.34,
//			}
//		}
//	}
func (db *PgDB) GetTrialProfilerMetricsBatches(
	labels *trialv1.TrialProfilerMetricLabels, offset, limit int,
) (model.TrialProfilerMetricsBatchBatch, error) {
	metricGroup := groupFromLabelName(labels.Name)

	if metricGroup == gpu && labels.GpuUuid == "" {
		// If GPU metrics are requested without a GPU UUID, the Web UI expects metrics for all
		// GPU UUIDs returned. This is done as a separate query to optimize fetch time.
		return db.getTrialProfilerMetricsAllGPUs(labels.Name, labels.AgentId, labels.TrialId, offset, limit)
	}

	// Translate old schema to new, which formats metrics with label IDs as keys.
	jsonPath := "metrics"
	if labels.AgentId != "" {
		jsonPath += fmt.Sprintf("->'%s'", labels.AgentId)
		if metricGroup == gpu && labels.GpuUuid != "" {
			jsonPath += fmt.Sprintf("->'%s'", labels.GpuUuid)
		}
	}

	rows, err := db.sql.Queryx(`
SELECT
    `+jsonPath+`->>$1 AS values,
    end_time AS timestamps
FROM system_metrics
WHERE trial_id=$2
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

// getTrialProfilerMetricsAllGPUs gets GPU metrics for all GPU UUIDs for a given agent and metric
// name. This method is a special case of GetTrialProfilerMetricsBatches for when the metric
// requested is a GPU metric, but no GPU UUID was specified. In this case, the Web UI expects
// metrics for all GPU UUIDs to be returned. It is implemented as a separate query to optimize
// fetch times.
// This method is slated for deprecation after the Web UI transitions to using generic metrics APIs.
func (db *PgDB) getTrialProfilerMetricsAllGPUs(
	metricName, agentID string, trialID int32, offset, limit int,
) (model.TrialProfilerMetricsBatchBatch, error) {
	var pBatches []*trialv1.TrialProfilerMetricsBatch

	metricGroup := groupFromLabelName(metricName)
	if metricGroup != gpu {
		return pBatches, nil
	}

	// Use all keys of parent object for all GPU UUIDs.
	allGPUKeys := fmt.Sprintf("jsonb_object_keys(metrics->'%s')", agentID)
	jsonPath := fmt.Sprintf("metrics->'%s'->%s", agentID, allGPUKeys)

	rows, err := db.sql.Queryx(`
SELECT
	`+allGPUKeys+` AS gpu,
    `+jsonPath+`->>$1 AS value,
    end_time AS timestamp
FROM system_metrics
WHERE trial_id=$2
AND metric_group='gpu'
ORDER by id
OFFSET $3 LIMIT $4`, metricName, trialID, offset, limit)
	if err != nil {
		return nil, err
	}
	var res struct {
		Gpu       *string
		Value     *float32
		Timestamp *time.Time
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.StructScan(&res); err != nil {
			return nil, errors.Wrap(err, "querying profiler metric batch")
		}

		if res.Value == nil {
			// This should never happen but in the case that one report doesn't contain all metric
			// keys, just skip it.
			continue
		}
		gpuUUID := ""
		if res.Gpu != nil {
			gpuUUID = *res.Gpu
		}
		pBatch := &trialv1.TrialProfilerMetricsBatch{
			Values:     []float32{*res.Value},
			Timestamps: []*timestamppb.Timestamp{protoutils.ToTimestamp(*res.Timestamp)},
			Labels: &trialv1.TrialProfilerMetricLabels{
				TrialId:    trialID,
				AgentId:    agentID,
				GpuUuid:    gpuUUID,
				Name:       metricName,
				MetricType: trialv1.TrialProfilerMetricLabels_PROFILER_METRIC_TYPE_SYSTEM,
			},
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
func GetTrialProfilerAvailableSeries(
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
		for agentID, aM := range m.Metrics {
			// Top-level keys are always Agent IDs.
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
