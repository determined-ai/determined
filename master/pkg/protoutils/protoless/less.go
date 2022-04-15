// Package protoless provides some helpers for comparing specific proto objects. It is a bridge
// between needing more complex sorting (not just on top-level proto keys, like apiServer.sort) and
// pushing queries into Bun (which takes more work).
package protoless

import (
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/checkpointv1"
)

func CheckpointLatestBatchLess(a, b *checkpointv1.Checkpoint) bool {
	l1, ok := a.Metadata.AsMap()[model.LatestBatchMetadataKey].(float64)
	if !ok {
		// Just consider missing as always lower.
		return true
	}
	l2, ok := b.Metadata.AsMap()[model.LatestBatchMetadataKey].(float64)
	if !ok {
		return false
	}
	if l1 == l2 {
		return CheckpointReportTimeLess(a, b)
	}
	return l1 < l2
}

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

func CheckpointReportTimeLess(a, b *checkpointv1.Checkpoint) bool {
	return a.ReportTime.AsTime().Before(b.ReportTime.AsTime())
}
