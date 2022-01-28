package job

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/proto/pkg/jobv1"
)

func TestMoveMessagesPromote(t *testing.T) {
	jobs := []*jobv1.Job{
		{JobId: "job0", Summary: &jobv1.JobSummary{}},
		{JobId: "job1", Summary: &jobv1.JobSummary{}},
		{JobId: "job2", Summary: &jobv1.JobSummary{}},
		{JobId: "job3", Summary: &jobv1.JobSummary{}},
	}
	targetIdx := 2
	anchorIdx := 1
	aheadOf := true
	expectedAnchor2 := jobs[0].JobId
	prioMsg, moveMsg, err := moveJobMessages(jobs, jobs[targetIdx], jobs[anchorIdx],
		anchorIdx, aheadOf)
	assert.NilError(t, err)
	assert.Assert(t, prioMsg == nil, "moveJobMessages should not return a priority message")
	assert.Assert(t, moveMsg != nil, "moveJobMessages should return a move message")
	assert.Equal(t, moveMsg.ID.String(), jobs[targetIdx].JobId,
		"moveJobMessages should return the correct target job id")
	assert.Equal(t, moveMsg.Anchor1.String(), jobs[anchorIdx].JobId,
		"moveJobMessages should return the correct anchor id")
	assert.Equal(t, moveMsg.Anchor2.String(), expectedAnchor2,
		"moveJobMessages should return the correct anchor id")
}

func TestMoveMessagesDemote(t *testing.T) {
	jobs := []*jobv1.Job{
		{JobId: "job0", Summary: &jobv1.JobSummary{}},
		{JobId: "job1", Summary: &jobv1.JobSummary{}},
		{JobId: "job2", Summary: &jobv1.JobSummary{}},
		{JobId: "job3", Summary: &jobv1.JobSummary{}},
	}
	targetIdx := 1
	anchorIdx := 2
	aheadOf := false
	expectedAnchor2 := jobs[3].JobId
	prioMsg, moveMsg, err := moveJobMessages(jobs, jobs[targetIdx], jobs[anchorIdx],
		anchorIdx, aheadOf)
	assert.NilError(t, err)
	assert.Assert(t, prioMsg == nil, "moveJobMessages should not return a priority message")
	assert.Assert(t, moveMsg != nil, "moveJobMessages should return a move message")
	assert.Equal(t, moveMsg.ID.String(), jobs[targetIdx].JobId,
		"moveJobMessages should return the correct target job id")
	assert.Equal(t, moveMsg.Anchor1.String(), jobs[anchorIdx].JobId,
		"moveJobMessages should return the correct anchor id")
	assert.Equal(t, moveMsg.Anchor2.String(), expectedAnchor2,
		"moveJobMessages should return the correct anchor id")
}

func TestMoveMessagesDemoteTail(t *testing.T) {
	jobs := []*jobv1.Job{
		{JobId: "job0", Summary: &jobv1.JobSummary{}},
		{JobId: "job1", Summary: &jobv1.JobSummary{}},
		{JobId: "job2", Summary: &jobv1.JobSummary{}},
		{JobId: "job3", Summary: &jobv1.JobSummary{}},
	}
	targetIdx := 1
	anchorIdx := 3
	aheadOf := false
	expectedAnchor2 := TailAnchor.String()
	prioMsg, moveMsg, err := moveJobMessages(jobs, jobs[targetIdx], jobs[anchorIdx],
		anchorIdx, aheadOf)
	assert.NilError(t, err)
	assert.Assert(t, prioMsg == nil, "moveJobMessages should not return a priority message")
	assert.Assert(t, moveMsg != nil, "moveJobMessages should return a move message")
	assert.Equal(t, moveMsg.ID.String(), jobs[targetIdx].JobId,
		"moveJobMessages should return the correct target job id")
	assert.Equal(t, moveMsg.Anchor1.String(), jobs[anchorIdx].JobId,
		"moveJobMessages should return the correct anchor id")
	assert.Equal(t, moveMsg.Anchor2.String(), expectedAnchor2,
		"moveJobMessages should return the correct anchor id")
}

func TestMoveMessagesPromoteHead(t *testing.T) {
	jobs := []*jobv1.Job{
		{JobId: "job0", Summary: &jobv1.JobSummary{}},
		{JobId: "job1", Summary: &jobv1.JobSummary{}},
		{JobId: "job2", Summary: &jobv1.JobSummary{}},
		{JobId: "job3", Summary: &jobv1.JobSummary{}},
	}
	targetIdx := 3
	anchorIdx := 0
	aheadOf := true
	expectedAnchor2 := HeadAnchor.String()
	prioMsg, moveMsg, err := moveJobMessages(jobs, jobs[targetIdx], jobs[anchorIdx],
		anchorIdx, aheadOf)
	assert.NilError(t, err)
	assert.Assert(t, prioMsg == nil, "moveJobMessages should not return a priority message")
	assert.Assert(t, moveMsg != nil, "moveJobMessages should return a move message")
	assert.Equal(t, moveMsg.ID.String(), jobs[targetIdx].JobId,
		"moveJobMessages should return the correct target job id")
	assert.Equal(t, moveMsg.Anchor1.String(), jobs[anchorIdx].JobId,
		"moveJobMessages should return the correct anchor id")
	assert.Equal(t, moveMsg.Anchor2.String(), expectedAnchor2,
		"moveJobMessages should return the correct anchor id")
}

func TestMoveMessagesSameJobID(t *testing.T) {
	jobs := []*jobv1.Job{
		{JobId: "job0", Summary: &jobv1.JobSummary{}},
		{JobId: "job1", Summary: &jobv1.JobSummary{}},
		{JobId: "job2", Summary: &jobv1.JobSummary{}},
		{JobId: "job3", Summary: &jobv1.JobSummary{}},
	}
	targetIdx := 0
	anchorIdx := 0
	aheadOf := true
	prioMsg, moveMsg, err := moveJobMessages(jobs, jobs[targetIdx], jobs[anchorIdx],
		anchorIdx, aheadOf)
	assert.NilError(t, err)
	assert.Assert(t, prioMsg == nil, "moveJobMessages should not return a priority message")
	assert.Assert(t, moveMsg == nil, "moveJobMessages should not return a move message")
}

func TestMoveMessagesSameEventualPosition(t *testing.T) {
	jobs := []*jobv1.Job{
		{JobId: "job0", Summary: &jobv1.JobSummary{}},
		{JobId: "job1", Summary: &jobv1.JobSummary{}},
		{JobId: "job2", Summary: &jobv1.JobSummary{}},
		{JobId: "job3", Summary: &jobv1.JobSummary{}},
	}
	targetIdx := 0
	anchorIdx := 1
	aheadOf := true
	prioMsg, moveMsg, err := moveJobMessages(jobs, jobs[targetIdx], jobs[anchorIdx],
		anchorIdx, aheadOf)
	assert.NilError(t, err)
	assert.Assert(t, prioMsg == nil, "moveJobMessages should not return a priority message")
	assert.Assert(t, moveMsg == nil, "moveJobMessages should not return a move message")
}

func TestMoveMessagesSameEventualPositionBehind(t *testing.T) {
	jobs := []*jobv1.Job{
		{JobId: "job0", Summary: &jobv1.JobSummary{}},
		{JobId: "job1", Summary: &jobv1.JobSummary{}},
		{JobId: "job2", Summary: &jobv1.JobSummary{}},
		{JobId: "job3", Summary: &jobv1.JobSummary{}},
	}
	targetIdx := 1
	anchorIdx := 0
	aheadOf := false
	prioMsg, moveMsg, err := moveJobMessages(jobs, jobs[targetIdx], jobs[anchorIdx],
		anchorIdx, aheadOf)
	assert.NilError(t, err)
	assert.Assert(t, prioMsg == nil, "moveJobMessages should not return a priority message")
	assert.Assert(t, moveMsg == nil, "moveJobMessages should not return a move message")
}

func TestMoveMessagesAcrossPrioLanes(t *testing.T) {
	jobs := []*jobv1.Job{
		{JobId: "job0", Priority: 1, Summary: &jobv1.JobSummary{}},
		{JobId: "job1", Priority: 1, Summary: &jobv1.JobSummary{}},
		{JobId: "job2", Priority: 2, Summary: &jobv1.JobSummary{}},
		{JobId: "job3", Priority: 2, Summary: &jobv1.JobSummary{}},
	}
	targetIdx := 0
	anchorIdx := 1
	aheadOf := false
	expectedAnchor2 := TailAnchor.String()
	prioMsg, moveMsg, err := moveJobMessages(jobs, jobs[targetIdx], jobs[anchorIdx],
		anchorIdx, aheadOf)
	assert.NilError(t, err)
	assert.Assert(t, prioMsg == nil, "moveJobMessages should not return a priority message")
	assert.Assert(t, moveMsg != nil, "moveJobMessages should return a move message")
	assert.Equal(t, moveMsg.ID.String(), jobs[targetIdx].JobId,
		"moveJobMessages should return the correct target job id")
	assert.Equal(t, moveMsg.Anchor1.String(), jobs[anchorIdx].JobId,
		"moveJobMessages should return the correct anchor id")
	assert.Equal(t, moveMsg.Anchor2.String(), expectedAnchor2,
		"moveJobMessages should return the correct anchor id")
}

func TestMoveMessagesAcrossPrioLanesAhead(t *testing.T) {
	jobs := []*jobv1.Job{
		{JobId: "job0", Priority: 1, Summary: &jobv1.JobSummary{}},
		{JobId: "job1", Priority: 1, Summary: &jobv1.JobSummary{}},
		{JobId: "job2", Priority: 2, Summary: &jobv1.JobSummary{}},
		{JobId: "job3", Priority: 2, Summary: &jobv1.JobSummary{}},
	}
	targetIdx := 2
	anchorIdx := 1
	aheadOf := true
	expectedAnchor2 := jobs[0].JobId
	prioMsg, moveMsg, err := moveJobMessages(jobs, jobs[targetIdx], jobs[anchorIdx],
		anchorIdx, aheadOf)
	assert.NilError(t, err)
	assert.Assert(t, prioMsg != nil, "moveJobMessages should return a priority message")
	assert.Equal(t, int32(prioMsg.Priority), jobs[anchorIdx].Priority,
		"moveJobMessages should return the correct priority")
	assert.Assert(t, moveMsg != nil, "moveJobMessages should return a move message")
	assert.Equal(t, moveMsg.ID.String(), jobs[targetIdx].JobId,
		"moveJobMessages should return the correct target job id")
	assert.Equal(t, moveMsg.Anchor1.String(), jobs[anchorIdx].JobId,
		"moveJobMessages should return the correct anchor id")
	assert.Equal(t, moveMsg.Anchor2.String(), expectedAnchor2,
		"moveJobMessages should return the correct anchor id")
}
