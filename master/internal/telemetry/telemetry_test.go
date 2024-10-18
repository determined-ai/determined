//go:build integration
// +build integration

package telemetry

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/segmentio/analytics-go.v3"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/device"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/resourcepoolv1"
)

var pgDB *db.PgDB

// TestMain sets up the DB for tests.
func TestMain(m *testing.M) {
	tmp, _, err := db.ResolveTestPostgres()
	pgDB = tmp
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	os.Exit(m.Run())
}

func TestTelemetry(t *testing.T) {
	// Mock out the telemetry actor & client interface.
	client, rm := initMockedTelemetry(t)

	// Should receive one master_tick event and identify initially, reset queue after check.
	reportMasterTick(pgDB, rm)
	assert.Equal(t, []string{"identify", "master_tick"}, client.getQueue(), "queue didn't receive initial master tick")
	client.resetQueue()

	// Test out Tick & reset the queue.
	delay := reportMasterTickDelay().Minutes()
	assert.True(t, (delay >= minTickIntervalMins) && (delay <= maxTickIntervalMins))

	// Test out Track & reset the queue.
	defaultTelemeter.track(analytics.Track{Event: "manual_call"})
	require.ElementsMatch(t, []string{"manual_call"}, client.getQueue(), "queue didn't receive correct track call")
	client.resetQueue()

	tIn := db.RequireMockTask(t, pgDB, nil)
	aIn := db.RequireMockAllocation(t, pgDB, tIn.TaskID)

	// Test out all Reports.
	reportMasterTick(pgDB, rm)
	ReportProvisionerTick([]*model.Instance{}, "test-instance")
	ReportExperimentCreated(1, schemas.WithDefaults(createExpConfig()))
	ReportAllocationTerminal(*aIn, &device.Device{})
	ReportExperimentStateChanged(pgDB, &model.Experiment{})
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
	require.ElementsMatch(t, expected, client.getQueue(), "queue didn't receive track calls in the right order")
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
func initMockedTelemetry(t *testing.T) (*mockClient, *mocks.ResourceManager) {
	mockRM := &mocks.ResourceManager{}
	mockRM.On("GetResourcePools", mock.Anything, mock.Anything).Return(
		&apiv1.GetResourcePoolsResponse{ResourcePools: []*resourcepoolv1.ResourcePool{}},
		nil,
	)

	client := &mockClient{}
	telemeter, err := newTelemeter(client, "1")
	require.NoError(t, err)
	defaultTelemeter = telemeter

	return client, mockRM
}

// Helper function for ReportExperimentCreated.
func createExpConfig() expconf.ExperimentConfig {
	//nolint:exhaustruct
	activeConfig := expconf.ExperimentConfig{
		RawSearcher: &expconf.SearcherConfig{
			RawMetric:       ptrs.Ptr("loss"),
			RawSingleConfig: &expconf.SingleConfig{},
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
