package searcher

import (
	"fmt"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSimulate(t *testing.T) {
	maxConcurrentTrials := 5
	maxTrials := 10
	divisor := 3.0
	maxTime := 900
	config := expconf.SearcherConfig{
		RawAdaptiveASHAConfig: &expconf.AdaptiveASHAConfig{
			RawMaxTime:             &maxTime,
			RawDivisor:             &divisor,
			RawMaxConcurrentTrials: &maxConcurrentTrials,
			RawMaxTrials:           &maxTrials,
			RawTimeMetric:          ptrs.Ptr("batches"),
			RawMode:                expconf.AdaptiveModePtr(expconf.StandardMode),
		},
		RawMetric:          ptrs.Ptr("loss"),
		RawSmallerIsBetter: ptrs.Ptr(true),
	}
	config = schemas.WithDefaults(config)
	intHparam := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"x": expconf.Hyperparameter{RawIntHyperparameter: intHparam},
	}

	res, err := Simulate(config, hparams)
	require.NoError(t, err)
	// xxx: test this
	fmt.Println(res)
}
