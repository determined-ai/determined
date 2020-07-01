package searcher

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

func TestEventLog(t *testing.T) {
	rand := nprand.New(0)
	log := NewEventLog()

	trialIDs := []int{7, 11, 13, 17}

	for i, trialID := range trialIDs {
		create := NewCreate(rand, nil, model.TrialWorkloadSequencerType)

		assert.Equal(t, log.TrialsRequested, i)
		log.OperationsCreated(create)
		assert.Equal(t, log.TrialsRequested, i+1)

		// Check that mappings between request and trial IDs are added correctly.
		log.TrialCreated(create, trialID)
		assert.Equal(t, log.TrialIDs[create.RequestID], trialID)
		assert.Equal(t, log.RequestIDs[trialID], create.RequestID)
	}

	for i, trialID := range trialIDs {
		assert.Equal(t, len(log.inFlightWorkloads), 0)
		log.OperationsCreated(NewTrain(log.RequestIDs[trialID], 1, defaultBatchesPerStep))
		assert.Equal(t, len(log.inFlightWorkloads), 1)

		assert.Equal(t, len(log.completedWorkloads), i)
		log.WorkloadCompleted(CompletedMessage{
			Type: "WORKLOAD_COMPLETED",
			Workload: Workload{
				Kind:         RunStep,
				ExperimentID: 1,
				TrialID:      trialID,
				StepID:       1,
				NumBatches:   defaultBatchesPerStep,
			},
			RunMetrics: make(map[string]interface{}),
		})
		assert.Equal(t, len(log.completedWorkloads), i+1)

		// Check that closing trials counts them correctly.
		assert.Equal(t, log.TrialsClosed, i)
		log.TrialClosed(log.RequestIDs[trialID])
		assert.Equal(t, log.TrialsClosed, i+1)
	}

	// Check that shutting down sets the appropriate flag.
	assert.Assert(t, !log.Shutdown)
	log.OperationsCreated(NewShutdown())
	assert.Assert(t, log.Shutdown)
}

func TestEventLogCheckpointCaching(t *testing.T) {
	rand := nprand.New(0)
	log := NewEventLog()

	trialID := 1
	stepID := 1

	create := NewCreate(rand, nil, model.TrialWorkloadSequencerType)
	log.TrialCreated(create, trialID)

	checkpointOperation := WorkloadOperation{
		Kind:      CheckpointModel,
		RequestID: create.RequestID,
		StepID:    stepID,
	}

	completedMessage := CompletedMessage{
		Type: "WORKLOAD_COMPLETED",
		Workload: Workload{
			Kind:         CheckpointModel,
			ExperimentID: 1,
			TrialID:      trialID,
			StepID:       stepID,
		},
	}

	_, ok := log.completedWorkloads[checkpointOperation]
	assert.Assert(t, !ok)
	log.uncommitted = nil

	// Check that an unrequested CheckpointModel CompletedMessage is logged but not acted upon.
	assert.Assert(t, !log.WorkloadCompleted(completedMessage))
	completed, ok := log.completedWorkloads[checkpointOperation]
	assert.Assert(t, ok)
	assert.Assert(t, !completed)
	assert.Equal(t, len(log.uncommitted), 1)
	log.uncommitted = nil

	// Check that another CompletedMessage has no effect.
	assert.Assert(t, !log.WorkloadCompleted(completedMessage))
	assert.Assert(t, !log.completedWorkloads[checkpointOperation])
	assert.Equal(t, len(log.uncommitted), 0)

	// Check that FilterCompletedCheckpoints catches the already-completed checkpoint.
	filteredOps, replayMsgs := log.FilterCompletedCheckpoints([]Operation{checkpointOperation})
	assert.Equal(t, len(filteredOps), 0)
	assert.Equal(t, len(replayMsgs), 1)
	assert.DeepEqual(t, completedMessage, replayMsgs[0])

	// Tell the EventLog the SearchMethod has now asked for the CheckpointModel operation.
	log.OperationsCreated(checkpointOperation)
	log.uncommitted = nil

	// Check that the next call to WorkloadCompleted returns true, that EventLog.completedWorkloads
	// is updated, and that there is no duplicated searcher event was created.
	assert.Assert(t, log.WorkloadCompleted(completedMessage))
	assert.Assert(t, log.completedWorkloads[checkpointOperation])
	assert.Equal(t, len(log.uncommitted), 0)

	// Check that another CompletedMessage has no effect
	assert.Assert(t, !log.WorkloadCompleted(completedMessage))
	assert.Assert(t, log.completedWorkloads[checkpointOperation])
	assert.Equal(t, len(log.uncommitted), 0)

	// Check that a requested CheckpointModel message is both logged and acted upon the first time.
	stepID++
	checkpointOperation = WorkloadOperation{
		Kind:      CheckpointModel,
		RequestID: create.RequestID,
		StepID:    stepID,
	}

	completedMessage = CompletedMessage{
		Type: "WORKLOAD_COMPLETED",
		Workload: Workload{
			Kind:         CheckpointModel,
			ExperimentID: 1,
			TrialID:      trialID,
			StepID:       stepID,
		},
	}

	log.OperationsCreated(checkpointOperation)
	log.uncommitted = nil
	_, ok = log.completedWorkloads[checkpointOperation]
	assert.Assert(t, !ok)

	assert.Assert(t, log.WorkloadCompleted(completedMessage))
	assert.Assert(t, log.completedWorkloads[checkpointOperation])
	assert.Equal(t, len(log.uncommitted), 1)
}
