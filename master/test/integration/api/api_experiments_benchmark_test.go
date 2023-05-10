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
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commonv1"
	"github.com/determined-ai/determined/proto/pkg/experimentv1"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/ptrs"

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
		Labels        []string
	}

	{
		for intSort, stringSort := range apiv1.GetExperimentsRequest_SortBy_name {
			for intOrder, stringOrder := range apiv1.OrderBy_name {
				testName := fmt.Sprintf("GetExperiments SortBy=%s OrderBy=%s",
					stringSort, stringOrder)
				RunBenchmark(t, fasterThan, testName, func(b *testing.B) {
					for i := 0; i < b.N; i++ {
						_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
							SortBy:        apiv1.GetExperimentsRequest_SortBy(intSort),
							OrderBy:       apiv1.OrderBy(intOrder),
							Limit:         maxWebLimit,
							ShowTrialData: true,
						})
						require.NoError(t, err)
					}
				})
			}
		}
	}

	{
		RunBenchmark(t, fasterThan, "GetExperiments Description=a", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Description:   "a",
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		RunBenchmark(t, fasterThan, "GetExperiments Description=longStringOfAs", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Description:   longStringOfAs,
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		var longestPresentDescription []expParams
		require.NoError(t, db.Bun().NewSelect().Model(&longestPresentDescription).
			ColumnExpr("config->>'description' AS description").
			OrderExpr("length(config->>'description') DESC").
			Limit(1).
			Scan(ctx))
		desc := "only for no experiments case"
		if len(longestPresentDescription) > 0 {
			desc = longestPresentDescription[0].Description
		}
		RunBenchmark(t, fasterThan, "GetExperiments Description=longestPresent", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Description:   desc,
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})
	}

	{
		RunBenchmark(t, fasterThan, "GetExperiments Name=a", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Name:          "a",
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		RunBenchmark(t, fasterThan, "GetExperiments Name=longStringOfAs", func(b *testing.B) {
			_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
				Name:          longStringOfAs,
				Limit:         maxWebLimit,
				ShowTrialData: true,
			})
			require.NoError(t, err)
		})

		var longestNameDescription []expParams
		require.NoError(t, db.Bun().NewSelect().Model(&longestNameDescription).
			ColumnExpr("config->>'name' AS name").
			OrderExpr("length(config->>'Name') DESC").
			Limit(1).
			Scan(ctx))
		name := "only for no experiments"
		if len(longestNameDescription) > 0 {
			name = longestNameDescription[0].Name
		}
		RunBenchmark(t, fasterThan, "GetExperiments Name=longestPresentName", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Name:          name,
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})
	}

	{
		RunBenchmark(t, fasterThan, "GetExperiments Labels=a", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Labels:        []string{"a"},
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		var aToZ []string
		for i := byte('a'); i <= byte('z'); i++ {
			aToZ = append(aToZ, string(i))
		}
		RunBenchmark(t, fasterThan, "GetExperiments Labels=aToZ", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Labels:        aToZ,
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		resp, err := api.GetExperimentLabels(ctx, &apiv1.GetExperimentLabelsRequest{})
		require.NoError(t, err)
		RunBenchmark(t, fasterThan, "GetExperiments Labels=mostPopularLabel", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Labels:        []string{append(resp.Labels, "only for when no labels exist")[0]},
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})
	}

	{
		RunBenchmark(t, fasterThan, "GetExperiments Archived=true", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Archived: wrapperspb.Bool(true),
					Limit:    maxWebLimit,
				})
				require.NoError(t, err)
			}
		})
		RunBenchmark(t, fasterThan, "GetExperiments Archived=false", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Archived:      wrapperspb.Bool(false),
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})
	}

	{
		var allStates []experimentv1.State
		for stateInt, stateString := range experimentv1.State_name {
			if stateString == "STATE_UNSPECIFIED" ||
				stateString == "STATE_STARTING" ||
				stateString == "STATE_RUNNING" ||
				stateString == "STATE_DELETED" ||
				stateString == "STATE_PULLING" ||
				stateString == "STATE_QUEUED" {
				continue // TODO bug in getexperiments
			}

			allStates = append(allStates, experimentv1.State(stateInt))
			testName := fmt.Sprintf("GetExperiments States=%s", stateString)
			RunBenchmark(t, fasterThan, testName, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
						States:        []experimentv1.State{experimentv1.State(stateInt)},
						Limit:         maxWebLimit,
						ShowTrialData: true,
					})
					require.NoError(t, err)
				}
			})
		}

		RunBenchmark(t, fasterThan, "GetExperiments States=allStates", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					States:        allStates,
					Limit:         maxWebLimit,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})
	}

	userResp, err := api.GetUsers(ctx, &apiv1.GetUsersRequest{})
	require.NoError(t, err)

	{
		RunBenchmark(t, fasterThan, "GetExperiments Users=admin,determined", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Limit:         maxWebLimit,
					Users:         []string{"admin", "determined"},
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		var allUsers []string
		for _, u := range userResp.Users {
			allUsers = append(allUsers, u.Username)
		}
		RunBenchmark(t, fasterThan, "GetExperiments Users=allUsers", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Limit:         maxWebLimit,
					Users:         allUsers,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})
	}

	{
		RunBenchmark(t, fasterThan, "GetExperiments UserIds=1,2", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Limit:         maxWebLimit,
					UserIds:       []int32{1, 2},
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		var allUsers []int32
		for _, u := range userResp.Users {
			allUsers = append(allUsers, u.Id)
		}
		RunBenchmark(t, fasterThan, "GetExperiments UserIds=allUsers", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Limit:         maxWebLimit,
					UserIds:       allUsers,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})
	}

	{
		RunBenchmark(t, fasterThan, "GetExperiments ProjectId=0", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Limit:         maxWebLimit,
					ProjectId:     0,
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})
	}

	{
		RunBenchmark(t, fasterThan, "GetExperiments ExperimentIdFilter=Lt1000", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Limit: maxWebLimit,
					ExperimentIdFilter: &commonv1.Int32FieldFilter{
						Lt: ptrs.Ptr(int32(1000)),
					},
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		RunBenchmark(t, fasterThan, "GetExperiments ExperimentIdFilter=Lte1000", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Limit: maxWebLimit,
					ExperimentIdFilter: &commonv1.Int32FieldFilter{
						Lte: ptrs.Ptr(int32(1000)),
					},
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		RunBenchmark(t, fasterThan, "GetExperiments ExperimentIdFilter=Gt0", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Limit: maxWebLimit,
					ExperimentIdFilter: &commonv1.Int32FieldFilter{
						Gt: ptrs.Ptr(int32(0)),
					},
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		RunBenchmark(t, fasterThan, "GetExperiments ExperimentIdFilter=Gte0", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
					Limit: maxWebLimit,
					ExperimentIdFilter: &commonv1.Int32FieldFilter{
						Gte: ptrs.Ptr(int32(0)),
					},
					ShowTrialData: true,
				})
				require.NoError(t, err)
			}
		})

		var oneToOneThousand []int32
		for i := int32(1); i <= 1000; i++ {
			oneToOneThousand = append(oneToOneThousand, i)
		}
		RunBenchmark(t, fasterThan, "GetExperiments ExperimentIdFilter=Incl(1-1000)",
			func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
						Limit: maxWebLimit,
						ExperimentIdFilter: &commonv1.Int32FieldFilter{
							Incl: oneToOneThousand,
						},
						ShowTrialData: true,
					})
					require.NoError(t, err)
				}
			})

		var twoToThreeThousand []int32
		for i := int32(2000); i <= 3000; i++ {
			twoToThreeThousand = append(twoToThreeThousand, i)
		}
		RunBenchmark(t, fasterThan, "GetExperiments ExperimentIdFilter=NotIn(2000-3000)",
			func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					_, err := api.GetExperiments(ctx, &apiv1.GetExperimentsRequest{
						Limit: maxWebLimit,
						ExperimentIdFilter: &commonv1.Int32FieldFilter{
							NotIn: twoToThreeThousand,
						},
						ShowTrialData: true,
					})
					require.NoError(t, err)
				}
			})
	}
}
