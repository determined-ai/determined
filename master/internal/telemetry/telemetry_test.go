package telemetry

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/config"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

var queue []string

func TestTelemetry(t *testing.T) {
	// Mock out the telemetry actor & client interface.
	InitTelemetryWithMocks()
	assert.NotNil(t, DefaultTelemeter)
	DefaultTelemeter.client = mockClient{}

	// Should receive one master_tick event during Init(), reset queue after check.
	time.Sleep(time.Second)
	assert.Equal(t, []string{"master_tick"},
		queue, "queue didn't receive initial master tick")
	queue = []string{}

	// Test out Tick & reset the queue.
	go DefaultTelemeter.tick(actor.NewSystem("Testing"))
	time.Sleep(time.Second)
	assert.Equal(t, []string{"master_tick"}, queue,
		"queue didn't receive test tick")
	queue = []string{}

	// Test out Track & reset the queue.
	DefaultTelemeter.track(analytics.Track{Event: "manual_call"})
	time.Sleep(time.Second)
	assert.Equal(t, []string{"manual_call"}, queue,
		"queue didn't receive correct track call")
	queue = []string{}

	// Test out all Reports.
	ReportMasterTick(&apiv1.GetResourcePoolsResponse{}, DefaultTelemeter.db)
	ReportProvisionerTick([]*model.Instance{}, "test-instance")
	ReportExperimentCreated(1, schemas.WithDefaults(createExpConfig()))
	ReportAllocationTerminal(DefaultTelemeter.db, model.Allocation{}, &device.Device{})
	ReportExperimentStateChanged(&db.PgDB{}, &model.Experiment{})
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
	assert.Equal(t, expected, queue, "queue didn't receive track calls in the right order")
}

type mockClient struct{}

func (m mockClient) Close() error {
	return nil
}

func (m mockClient) Enqueue(msg analytics.Message) error {
	switch ms := msg.(type) {
	case analytics.Identify:
		queue = append(queue, "identify")
	case analytics.Track:
		queue = append(queue, ms.Event)
	default:
		err := fmt.Errorf("messages with custom types cannot be enqueued: %T", msg)
		return err
	}
	return nil
}

// InitTelemetryWithMocks() calls Init(), which calls New().
func InitTelemetryWithMocks() {
	mockRM := &mocks.ResourceManager{}
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).Return(
		&apiv1.GetResourcePoolsResponse{ResourcePools: []*resourcepoolv1.ResourcePool{}},
		nil,
	)
	mockDB := &mocks.DB{}
	mockDB.On("PeriodicTelemetryInfo").Return([]byte(`{"master_version": 1}`), nil)
	mockDB.On("CompleteAllocationTelemetry", mock.Anything).Return([]byte(`{"allocation_id": 1}`), nil)
	Init(actor.NewSystem("Testing"), mockDB, mockRM, "1",
		config.TelemetryConfig{Enabled: true, SegmentMasterKey: "Test"},
	)
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
