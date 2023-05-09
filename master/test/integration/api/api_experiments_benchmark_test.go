//go:build integration
// +build integration

package api

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	//"github.com/determined-ai/determined/proto/pkg/commonv1"
	//"github.com/determined-ai/determined/proto/pkg/trialv1"
	//"github.com/determined-ai/determined/master/internal"
	//"github.com/determined-ai/determined/master/internal/db"
	//"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/apiv1"

	"github.com/determined-ai/determined/master/internal/db"

	"github.com/determined-ai/determined/master/test/testutils"
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
	maxWebLimit := int32(200)

	// TODO don't do this everytime.
	ctx := context.Background()
	_, _, api, ctx, err := testutils.RunMaster(ctx, nil)
	require.NoError(t, err)

	longBytesOfAs := make([]byte, 1024*1024)
	for i := range longBytesOfAs {
		longBytesOfAs[i] = 'a'
	}
	longStringOfAs := string(longBytesOfAs)

	type expParams struct {
		bun.BaseModel `bun:"table:experiments"`
		Description   string
		Name          string
	}

	{
		for intSort, stringSort := range apiv1.GetExperimentsRequest_SortBy_name {
			for intOrder, stringOrder := range apiv1.OrderBy_name {
				testName := fmt.Sprintf("GetExperiments SortBy=%s OrderBy=%s", stringSort, stringOrder)
				RunBenchmark(t, fasterThan, testName, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
							SortBy:  apiv1.GetExperimentsRequest_SortBy(intSort),
							OrderBy: apiv1.OrderBy(intOrder),
							Limit:   maxWebLimit,
						})
						require.NoError(t, err)
					}
				})
			}
		}
	}

	{
		RunBenchmark(t, fasterThan, "GetExperiments Description=a", func(b *testing.B) {
			_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
				Description: "a",
				Limit:       maxWebLimit,
			})
			require.NoError(t, err)
		})

		RunBenchmark(t, fasterThan, "GetExperiments Description=longStringOfAs", func(b *testing.B) {
			_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
				Description: string(longStringOfAs),
				Limit:       maxWebLimit,
			})
			require.NoError(t, err)
		})

		longestPresentDescription := expParams{}
		require.NoError(t, db.Bun().NewSelect().Model(&longestPresentDescription).
			ColumnExpr("config->>'description' AS description").
			OrderExpr("length(config->>'description') DESC").
			Limit(1).
			Scan(ctx))
		RunBenchmark(t, fasterThan, "GetExperiments Description=longestPresent", func(b *testing.B) {
			_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
				Description: longestPresentDescription.Description,
				Limit:       maxWebLimit,
			})
			require.NoError(t, err)
		})
	}

	{
		RunBenchmark(t, fasterThan, "GetExperiments Name=a", func(b *testing.B) {
			_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
				Name:  "a",
				Limit: maxWebLimit,
			})
			require.NoError(t, err)
		})

		RunBenchmark(t, fasterThan, "GetExperiments Name=longStringOfAs", func(b *testing.B) {
			_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
				Name:  longStringOfAs,
				Limit: maxWebLimit,
			})
			require.NoError(t, err)
		})

		longestNameDescription := expParams{}
		require.NoError(t, db.Bun().NewSelect().Model(&longestNameDescription).
			ColumnExpr("config->>'name' AS name").
			OrderExpr("length(config->>'Name') DESC").
			Limit(1).
			Scan(ctx))
		RunBenchmark(t, fasterThan, "GetExperiments Name=longestPresentName", func(b *testing.B) {
			_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
				Name:  longestNameDescription.Name,
				Limit: maxWebLimit,
			})
			require.NoError(t, err)
		})
	}

	RunBenchmark(t, fasterThan, "GetExperiments Labels=%s", func(b *testing.B) {
	})

	RunBenchmark(t, fasterThan, "GetExperiments Archived=true", func(b *testing.B) {
	})
	RunBenchmark(t, fasterThan, "GetExperiments Archived=false", func(b *testing.B) {
	})

	RunBenchmark(t, fasterThan, "GetExperiments States=%s", func(b *testing.B) {
	})

	RunBenchmark(t, fasterThan, "GetExperiments Users=%s", func(b *testing.B) {
	})

	RunBenchmark(t, fasterThan, "GetExperiments UserIds=%s", func(b *testing.B) {
	})

	RunBenchmark(t, fasterThan, "GetExperiments ProjectId=%s", func(b *testing.B) {
	})

	RunBenchmark(t, fasterThan, "GetExperiments ExperimentIdFilter=%s", func(b *testing.B) {
	})

	RunBenchmark(t, fasterThan, "GetExperiments ShowTrialData=%s", func(b *testing.B) {
	})

	// TODOs filters...

	// for _, t := range
}
