//go:build integration
// +build integration

package internal

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/masterv1"
)

// TODO(DET-10109) test and optimize the monthly version.
func TestResourceAllocationAggregatedDaily(t *testing.T) {
	require.NoError(t, etc.SetRootPath("../static/srv"))

	pgDB, cleanup := db.MustResolveNewPostgresDatabase(t)
	defer cleanup()
	db.MustMigrateTestPostgres(t, pgDB, "file://../static/migrations")

	api, _, ctx := setupAPITest(t, pgDB)

	pgTable := []struct {
		bun.BaseModel   `bun:"table:resource_aggregates"`
		Date            string
		AggregationType string
		AggregationKey  string
		Seconds         float64
	}{
		{bun.BaseModel{}, "2020-09-02", "total", "total", 1.25},
		{bun.BaseModel{}, "2020-10-01", "total", "total", 2.25},
		{bun.BaseModel{}, "2020-11-19", "total", "total", 3.25},

		{bun.BaseModel{}, "2020-10-01", "username", "a", 1.5},
		{bun.BaseModel{}, "2020-10-01", "username", "b", 2.5},
		{bun.BaseModel{}, "2020-10-10", "username", "a", 3.5},
		{bun.BaseModel{}, "2020-10-10", "username", "c", 4.5},
		{bun.BaseModel{}, "2020-10-11", "username", "c", 5.5},
		{bun.BaseModel{}, "2020-11-20", "username", "c", 6.5},

		{bun.BaseModel{}, "2020-10-01", "experiment_label", "el_a", 1.75},
		{bun.BaseModel{}, "2020-10-01", "experiment_label", "el_b", 2.75},
		{bun.BaseModel{}, "2020-10-19", "experiment_label", "el_a", 3.75},
		{bun.BaseModel{}, "2020-11-19", "experiment_label", "el_a", 4.75},
		{bun.BaseModel{}, "2020-11-19", "experiment_label", "el_c", 5.75},

		{bun.BaseModel{}, "2020-10-01", "resource_pool", "rp_a", 1.0},
		{bun.BaseModel{}, "2020-10-01", "resource_pool", "rp_b", 2.0},
		{bun.BaseModel{}, "2020-10-18", "resource_pool", "rp_a", 3.0},
		{bun.BaseModel{}, "2020-11-19", "resource_pool", "rp_b", 4.0},
	}
	_, err := db.Bun().NewInsert().Model(&pgTable).Exec(ctx)
	require.NoError(t, err)

	cases := []struct {
		name     string
		start    string
		end      string
		expected string
	}{
		{"whole time", "2000-01-01", "2099-01-01", `[
  {
    "period_start": "2020-09-02",
    "period": 1,
    "seconds": 1.25
  },
  {
    "period_start": "2020-10-01",
    "period": 1,
    "seconds": 2.25,
    "by_username": {
      "a": 1.5,
      "b": 2.5
    },
    "by_experiment_label": {
      "el_a": 1.75,
      "el_b": 2.75
    },
    "by_resource_pool": {
      "rp_a": 1,
      "rp_b": 2
    }
  },
  {
    "period_start": "2020-10-10",
    "period": 1,
    "by_username": {
      "a": 3.5,
      "c": 4.5
    }
  },
  {
    "period_start": "2020-10-11",
    "period": 1,
    "by_username": {
      "c": 5.5
    }
  },
  {
    "period_start": "2020-10-18",
    "period": 1,
    "by_resource_pool": {
      "rp_a": 3
    }
  },
  {
    "period_start": "2020-10-19",
    "period": 1,
    "by_experiment_label": {
      "el_a": 3.75
    }
  },
  {
    "period_start": "2020-11-19",
    "period": 1,
    "seconds": 3.25,
    "by_experiment_label": {
      "el_a": 4.75,
      "el_c": 5.75
    },
    "by_resource_pool": {
      "rp_b": 4
    }
  },
  {
    "period_start": "2020-11-20",
    "period": 1,
    "by_username": {
      "c": 6.5
    }
  }
]`},

		{"just 10/01", "2020-10-01", "2020-10-01", `[
  {
    "period_start": "2020-10-01",
    "period": 1,
    "seconds": 2.25,
    "by_username": {
      "a": 1.5,
      "b": 2.5
    },
    "by_experiment_label": {
      "el_a": 1.75,
      "el_b": 2.75
    },
    "by_resource_pool": {
      "rp_a": 1,
      "rp_b": 2
    }
  }
]`},
		{"10/01 to 10/10", "2020-10-01", "2020-10-10", `[
  {
    "period_start": "2020-10-01",
    "period": 1,
    "seconds": 2.25,
    "by_username": {
      "a": 1.5,
      "b": 2.5
    },
    "by_experiment_label": {
      "el_a": 1.75,
      "el_b": 2.75
    },
    "by_resource_pool": {
      "rp_a": 1,
      "rp_b": 2
    }
  },
  {
    "period_start": "2020-10-10",
    "period": 1,
    "by_username": {
      "a": 3.5,
      "c": 4.5
    }
  }
]`},
		{"earlier than start", "2000-01-01", "2020-09-02", `[
  {
    "period_start": "2020-09-02",
    "period": 1,
    "seconds": 1.25
  }
]`},
		{"later than end", "2020-11-20", "2024-01-01", `[
  {
    "period_start": "2020-11-20",
    "period": 1,
    "by_username": {
      "c": 6.5
    }
  }
]`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, err := api.ResourceAllocationAggregated(ctx, &apiv1.ResourceAllocationAggregatedRequest{
				Period:    masterv1.ResourceAllocationAggregationPeriod_RESOURCE_ALLOCATION_AGGREGATION_PERIOD_DAILY,
				StartDate: c.start,
				EndDate:   c.end,
			})
			require.NoError(t, err)

			actual, err := json.MarshalIndent(resp.ResourceEntries, "", "  ")
			require.NoError(t, err)

			require.Equal(t, c.expected, string(actual))
		})
	}
}
