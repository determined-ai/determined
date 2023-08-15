package telemetry

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

type mockClient struct{}

var queue []string

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

func TestTelemetry(t *testing.T) {
	// Mock out the telemetry actor & client interface.
	// MockTelemetry() calls InitTelemetry(), which calls New().
	MockTelemetry()
	DefaultTelemetry.client = mockClient{}
	assert.NotNil(t, DefaultTelemetry)

	// Should receive one master_tick event during InitTelemetry(), reset queue after check.
	time.Sleep(time.Second)
	assert.Equal(t, 1, len(queue))
	assert.Equal(t, []string{"master_tick"}, queue)
	queue = []string{}

	// Test out Tick & reset the queue.
	DefaultTelemetry.telemetryTick(actor.NewSystem("Testing"), 0)
	time.Sleep(time.Second)
	assert.Equal(t, 1, len(queue))
	assert.Equal(t, []string{"master_tick"}, queue)
	queue = []string{}

	// Test out Track & reset the queue.
	DefaultTelemetry.Track(analytics.Track{Event: "manual_call"})
	time.Sleep(time.Second)
	assert.Equal(t, 1, len(queue))
	assert.Equal(t, []string{"manual_call"}, queue)
	queue = []string{}

	// Test out all Reports.
	ReportMasterTick(&apiv1.GetResourcePoolsResponse{}, DefaultTelemetry.db)
	ReportProvisionerTick([]*model.Instance{}, "test-instance")
	ReportExperimentCreated(1, schemas.WithDefaults(createExpConfig()))
	ReportAllocationTerminal(DefaultTelemetry.db, model.Allocation{}, &device.Device{})
	ReportExperimentStateChanged(&db.PgDB{}, model.Experiment{})
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
	assert.Equal(t, len(expected), len(queue))
	assert.Equal(t, expected, queue)
}
