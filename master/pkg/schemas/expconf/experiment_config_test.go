package expconf

import (
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/schemas"
)

func TestBindMountsMerge(t *testing.T) {
	e1 := ExperimentConfig{
		BindMounts: &BindMountsConfig{
			BindMount{
				HostPath:      "/host/e1",
				ContainerPath: "/container/e1",
			},
		},
	}
	e2 := ExperimentConfig{
		BindMounts: &BindMountsConfig{
			BindMount{
				HostPath:      "/host/e2",
				ContainerPath: "/container/e2",
			},
		},
	}
	schemas.Merge(&e1, e2)
	assert.Assert(t, len(*e1.BindMounts) == 2)
	assert.Assert(t, (*e1.BindMounts)[0].HostPath == "/host/e1")
	assert.Assert(t, (*e1.BindMounts)[1].HostPath == "/host/e2")
}

// XXX test trial counts somehow.

// // TestGridValidation tests that invalid grid search configurations produce validation errors and
// // valid ones don't.
// func TestGridValidation(t *testing.T) {
// 	// XXX: what was is this test even doing?
// 	// Check that counts for int hyperparameters are clamped properly.
// 	{
// 		config := validGridSearchConfig()
// 		config.Hyperparameters["log"].LogHyperparameter.Count = intP(1)
// 		config.Hyperparameters["int"].IntHyperparameter.Count = intP(100000)
// 	}
// }
