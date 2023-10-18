// Package protoless provides some helpers for comparing specific proto objects. It is a bridge
// between needing more complex sorting (not just on top-level proto keys, like apiServer.sort) and
// pushing queries into Bun (which takes more work).
package protoless

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

// CheckpointStepsCompletedLess compares checkpoints by their number of steps completed, falling
// back to report time when equal.
func CheckpointStepsCompletedLess(a, b *checkpointv1.Checkpoint) bool {
	l1, ok := a.Metadata.AsMap()[model.StepsCompletedMetadataKey].(float64)
	if !ok {
		// Just consider missing as always lower.
		return true
	}
	l2, ok := b.Metadata.AsMap()[model.StepsCompletedMetadataKey].(float64)
	if !ok {
		return false
	}
	if l1 == l2 {
		return CheckpointReportTimeLess(a, b)
	}
	return l1 < l2
}

// CheckpointTrialIDLess compares checkpoints by their ID, falling back to report time when equal.
func CheckpointTrialIDLess(ai, aj *checkpointv1.Checkpoint) bool {
	if ai.Training == nil || ai.Training.TrialId == nil {
		return true
	}
	if aj.Training == nil || aj.Training.TrialId == nil {
		return false
	}
	if ai.Training.TrialId.Value == aj.Training.TrialId.Value {
		return CheckpointReportTimeLess(ai, aj)
	}
	return ai.Training.TrialId.Value < aj.Training.TrialId.Value
}

// CheckpointSearcherMetricLess compares checkpoints by their searcher metric, falling back to
// report time when equal. Order makes sure nulls are always last.
func CheckpointSearcherMetricLess(ai, aj *checkpointv1.Checkpoint) bool {
	if ai.Training == nil || ai.Training.SearcherMetric == nil {
		return true
	}
	if aj.Training == nil || aj.Training.SearcherMetric == nil {
		return false
	}
	if ai.Training.SearcherMetric.Value == aj.Training.SearcherMetric.Value {
		return CheckpointReportTimeLess(ai, aj)
	}
	return ai.Training.SearcherMetric.Value < aj.Training.SearcherMetric.Value
}

// CheckpointMetricNameLess compares checkpoints by a metric name, falling back to
// report time when equal. Order makes sure nulls are always last.
func CheckpointMetricNameLess(ai, aj *checkpointv1.Checkpoint, metricName string) bool {
	aiMetricValue, aiOk := ai.GetTraining().GetValidationMetrics().GetAvgMetrics().AsMap()[metricName]
	ajMetricValue, ajOk := aj.GetTraining().GetValidationMetrics().GetAvgMetrics().AsMap()[metricName]
	if !aiOk {
		return true
	}
	if !ajOk {
		return false
	}
	if aiMetricValue == ajMetricValue {
		return CheckpointReportTimeLess(ai, aj)
	}
	return aiMetricValue.(float64) < ajMetricValue.(float64)
}

// CheckpointSearcherMetricNullsLast compares checkpoints by their searcher metric, return done if
// one was null and the proper ordering.
func CheckpointSearcherMetricNullsLast(ai, aj *checkpointv1.Checkpoint) (order bool, done bool) {
	if ai.Training == nil || ai.Training.SearcherMetric == nil {
		return false, true
	}
	if aj.Training == nil || aj.Training.SearcherMetric == nil {
		return true, true
	}
	return false, false
}

// CheckpointReportTimeLess compares checkpoints by their report time.
func CheckpointReportTimeLess(a, b *checkpointv1.Checkpoint) bool {
	return a.ReportTime.AsTime().Before(b.ReportTime.AsTime())
}
