package internal

import (
	"reflect"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

var rp = "mock"

func TestResourcePool(t *testing.T) {
	e := mockInternalExp()
	require.Equal(t, mutexLocked(&e.mu), false)

	rpName := e.ResourcePool()
	require.Equal(t, rp, rpName)
	require.Equal(t, mutexLocked(&e.mu), false)
}

func mutexLocked(m *sync.Mutex) bool {
	state := reflect.ValueOf(m).Elem().FieldByName("state")
	return state.Int()&1 == 1
}

func mockInternalExp() *internalExperiment {
	//nolint:exhaustruct
	return &internalExperiment{
		activeConfig: expconf.ExperimentConfigV0{
			RawResources: &expconf.ResourcesConfigV0{
				RawResourcePool: &rp,
			},
		},
	}
}
