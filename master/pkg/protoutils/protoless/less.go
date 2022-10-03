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

// CheckpointReportTimeLess compares checkpoints by their report time.
func CheckpointReportTimeLess(a, b *checkpointv1.Checkpoint) bool {
	return a.ReportTime.AsTime().Before(b.ReportTime.AsTime())
}
