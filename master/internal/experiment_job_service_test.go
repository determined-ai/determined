package internal

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestResourcePool(t *testing.T) {
	rp := "mock"
	//nolint:exhaustruct
	e := &internalExperiment{
		activeConfig: expconf.ExperimentConfigV0{
			RawResources: &expconf.ResourcesConfigV0{
				RawResourcePool: &rp,
			},
		},
	}

	rpName := e.ResourcePool()
	require.Equal(t, rp, rpName)
}
