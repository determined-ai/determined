//go:build integration
// +build integration

package telemetry

import (
	"testing"
)

/*
type mockRM struct{}
func (m *mockRM) GetResourcePools(

	actor.Messenger,
	*apiv1.GetResourcePoolsRequest,

	) (*apiv1.GetResourcePoolsResponse, error) {
		return *apiv1.GetResourcePoolsRequest{}, nil
	}

	func setup(t *testing.T) {
		InitTelemetry(
			actor.NewSystem(t.Name()),
			*db.PgDB{},
			&mockRM{}, "1",
			config.TelemetryConfig{
				Enabled:          true,
				SegmentMasterKey: masterSecret,
				SegmentWebUIKey:  webuiSecret,
			},
		)
	}
*/
func TestTelemetry(t *testing.T) {
	// TODO CAROLINA.
	// Test w/out InitTelemetry.
	// setup(t)
	// assert.NotNil(DefaultTelemetry)

	// Test w InitTelemetry.

	// Test out all Reports.

	// Check that client enqueue is correct.
}
