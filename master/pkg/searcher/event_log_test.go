package searcher

import (
	"github.com/determined-ai/determined/master/pkg/workload"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/nprand"
)

func TestEventLog(t *testing.T) {
	rand := nprand.New(0)
	log := NewEventLog(model.Batches)

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
		// Check that units completed are recorded and workloads saved.
		msg := workload.CompletedMessage{
			Workload: workload.Workload{
				StepID: trialID*trialID + i,
			},
		}
		assert.Equal(t, log.TotalUnitsCompleted, float64(i*10))
		log.WorkloadCompleted(msg, 10)
		assert.Equal(t, log.TotalUnitsCompleted, float64((i+1)*10))
	}

	assert.Equal(t, log.TotalUnitsCompleted, float64(10*len(trialIDs)))

	for i, trialID := range trialIDs {
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
