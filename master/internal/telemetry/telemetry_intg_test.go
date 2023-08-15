//go:build integration
// +build integration

package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setup(t *testing.T) {
	MockTelemetry()
}

func TestTelemetry(t *testing.T) {
	setup(t)
	assert.NotNil(t, DefaultTelemetry)

	// Test out all Reports.
	// ReportMasterTick(&apiv1.GetResourcePoolsResponse{}, &db.PgDB{})

	// ReportProvisionerTick([]*model.Instance{}, "test-instance")

	// ReportExperimentCreated(1, expconf.ExperimentConfig{})

	// gReportAllocationTerminal(&db.PgDB{}, a model.Allocation, d *device.Device,)

	// Check that client enqueue is correct.
}
