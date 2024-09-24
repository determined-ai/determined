//nolint:exhaustruct
package searcher

import (
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestRandomSearchMethod(t *testing.T) {
	conf := expconf.SearcherConfig{
		RawRandomConfig: &expconf.RandomConfig{
			RawMaxTrials:           ptrs.Ptr(4),
			RawMaxConcurrentTrials: ptrs.Ptr(2),
			RawMaxLength:           ptrs.Ptr(expconf.NewLengthInBatches(300)),
		},
	}
	conf = schemas.WithDefaults(conf)
	intHparam := &expconf.IntHyperparameter{RawMaxval: 10, RawCount: ptrs.Ptr(3)}
	hparams := expconf.Hyperparameters{
		"x": expconf.Hyperparameter{RawIntHyperparameter: intHparam},
	}
	testSearchRunner := NewTestSearchRunner(t, conf, hparams)
	created, stopped := testSearchRunner.start()
	require.Len(t, created, 2)
	require.Len(t, stopped, 0)
}

func TestSingleSearchMethod(t *testing.T) {
	// xxx: test this
}
