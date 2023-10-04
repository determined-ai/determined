package telemetry

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

func TestTelemetry(t *testing.T) {
	// Mock out the telemetry actor & client interface.
	client, rm, db, system := initMockedTelemetry(t)

	// Should receive one master_tick event and identify initially, reset queue after check.
	reportMasterTick(db, rm, system)
	assert.Equal(t, []string{"identify", "master_tick"}, client.getQueue(), "queue didn't receive initial master tick")
	client.resetQueue()

	// Test out Tick & reset the queue.
	delay := reportMasterTickDelay().Minutes()
	assert.True(t, (delay >= minTickIntervalMins) && (delay <= maxTickIntervalMins))

	// Test out Track & reset the queue.
	defaultTelemeter.track(analytics.Track{Event: "manual_call"})
	assert.Equal(t, []string{"manual_call"}, client.getQueue(), "queue didn't receive correct track call")
	client.resetQueue()

	// Test out all Reports.
	reportMasterTick(db, rm, system)
	ReportProvisionerTick([]*model.Instance{}, "test-instance")
	ReportExperimentCreated(1, schemas.WithDefaults(createExpConfig()))
	ReportAllocationTerminal(db, model.Allocation{}, &device.Device{})
	ReportExperimentStateChanged(db, &model.Experiment{})
	ReportUserCreated(true, true)
	ReportUserCreated(false, false)

	expected := []string{
		"master_tick",
		"provisioner_tick",
		"experiment_created",
		"allocation_terminal",
		"experiment_state_changed",
		"user_created",
		"user_created",
	}
	assert.Equal(t, expected, client.getQueue(), "queue didn't receive track calls in the right order")
}

type mockClient struct {
	queue []string
}

func (m *mockClient) Close() error {
	return nil
}

func (m *mockClient) Enqueue(msg analytics.Message) error {
	switch ms := msg.(type) {
	case analytics.Identify:
		m.queue = append(m.queue, "identify")
	case analytics.Track:
		m.queue = append(m.queue, ms.Event)
	default:
		err := fmt.Errorf("messages with custom types cannot be enqueued: %T", msg)
		return err
	}
	return nil
}

func (m *mockClient) getQueue() []string {
	return m.queue
}

func (m *mockClient) resetQueue() {
	m.queue = []string{}
}

// initMockedTelemetry() does what Init() does, but for tests.
func initMockedTelemetry(t *testing.T) (*mockClient, *mocks.ResourceManager, *mocks.DB, *actor.System) {
	mockRM := &mocks.ResourceManager{}
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).Return(
		&apiv1.GetResourcePoolsResponse{ResourcePools: []*resourcepoolv1.ResourcePool{}},
		nil,
	)
	mockDB := &mocks.DB{}
	mockDB.On("PeriodicTelemetryInfo").Return([]byte(`{"master_version": 1}`), nil)
	mockDB.On("CompleteAllocationTelemetry", mock.Anything).Return([]byte(`{"allocation_id": 1}`), nil)

	system := actor.NewSystem("Testing")

	client := &mockClient{}
	telemeter, err := newTelemeter(client, "1")
	require.NoError(t, err)
	defaultTelemeter = telemeter

	return client, mockRM, mockDB, system
}

// Helper function for ReportExperimentCreated.
func createExpConfig() expconf.ExperimentConfig {
	maxLength := expconf.NewLengthInBatches(100)
	activeConfig := expconf.ExperimentConfig{
		RawSearcher: &expconf.SearcherConfig{
			RawMetric: ptrs.Ptr("loss"),
			RawSingleConfig: &expconf.SingleConfig{
				RawMaxLength: &maxLength,
			},
		},
		RawEntrypoint:      &expconf.Entrypoint{RawEntrypoint: "model_def:SomeTrialClass"},
		RawHyperparameters: expconf.Hyperparameters{},
		RawCheckpointStorage: &expconf.CheckpointStorageConfig{
			RawSharedFSConfig: &expconf.SharedFSConfig{
				RawHostPath: ptrs.Ptr("/"),
			},
		},
	}
	return activeConfig
}
