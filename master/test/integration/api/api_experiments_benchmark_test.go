//go:build integration
// +build integration

package api

import (
	"fmt"
	"testing"
	"time"
	//"github.com/determined-ai/determined/proto/pkg/commonv1"
	//"github.com/determined-ai/determined/proto/pkg/trialv1"
	//"github.com/determined-ai/determined/master/internal"
	//"github.com/determined-ai/determined/master/internal/db"
	//"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// Thin wrapper over b.Run with an undertime that the test will fail.
func RunBenchmark(t *testing.T, fasterThanTime time.Duration, name string, f func(b *testing.B)) {
	start := time.Now().UnixNano()

	// TODO go might decide to run functions fewer than we or -- or more than we want.
	res := testing.Benchmark(f)
	// panic(res)
	perIterationTime := time.Duration(int64(res.T) / int64(res.N))
	if perIterationTime > fasterThanTime {
		t.Errorf("BenchResult(%d-%d) %s "+
			"failed benchmark took an average of %v but expected under %v (iterations=%d)",
			start, time.Now().UnixNano(), name, perIterationTime, fasterThanTime, res.N)
	} else {
		t.Logf("BenchResult(%d-%d) %s "+
			"passed benchmark took an average of %v expected under %v (iterations=%d)",
			start, time.Now().UnixNano(), name, perIterationTime, fasterThanTime, res.N)
	}
}

func TestBenchmarkGetExperiments(t *testing.T) {
	fasterThan := 500 * time.Millisecond

	/*
		_, _, cl, creds, err := testutils.RunMaster(ctx, nil)
		if err != nil {
			panic(err) // TODO
		}
	*/

	for intSort, stringSort := range apiv1.GetExperimentsRequest_SortBy_name {
		for intOrder, stringOrder := range apiv1.OrderBy_name {
			testName := fmt.Sprintf("GetExperiments SortBy=%s OrderBy=%s", stringSort, stringOrder)
			RunBenchmark(t, fasterThan, testName, func(b *testing.B) {
				time.Sleep(1 * time.Millisecond)
				if intSort == 10 {
					time.Sleep(1000 * time.Millisecond)
				}

				_, _ = intSort, intOrder
			})
		}
	}

	// TODOs filters...

	// for _, t := range
}
