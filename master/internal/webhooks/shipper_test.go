//go:build integration

package webhooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const (
	pathToMigrations = "file://../../static/migrations"
	subtestTimeout   = 10 * time.Second
)

func TestShipper(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Log("setup db")
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, pathToMigrations)

	t.Log("clear db")
	_, err := db.Bun().NewDelete().Model((*Webhook)(nil)).Where("1=1").Exec(ctx)
	require.NoError(t, err)
	_, err = db.Bun().NewDelete().Model((*Event)(nil)).Where("1=1").Exec(ctx)
	require.NoError(t, err)

	t.Log("setup test webhook receiver")
	received := make(chan model.Experiment, 100)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e model.Experiment
		if json.NewDecoder(r.Body).Decode(&e); err != nil {
			t.Logf("error reading webhook body: %v", err)
			t.FailNow()
			return
		}
		t.Logf("received event for experiment %d:%v", e.ID, e.State)
		received <- e
	}))
	defer ts.Close()

	t.Log("setup a few test webhooks")
	// One with two triggers so it fires twice.
	require.NoError(t, AddWebhook(ctx, &Webhook{
		URL: "localhost:8080",
		Triggers: []*Trigger{
			{
				TriggerType: TriggerTypeStateChange,
				Condition: map[string]interface{}{
					"state": model.CompletedState,
				},
			},
			{
				TriggerType: TriggerTypeStateChange,
				Condition: map[string]interface{}{
					"state": model.CompletedState,
				},
			},
		},
		WebhookType: WebhookTypeDefault,
	}))
	// And one that just fires once.
	require.NoError(t, AddWebhook(ctx, &Webhook{
		URL: "localhost:8080",
		Triggers: []*Trigger{
			{
				TriggerType: TriggerTypeStateChange,
				Condition: map[string]interface{}{
					"state": model.CompletedState,
				},
			},
		},
		WebhookType: WebhookTypeDefault,
	}))

	t.Log("build shipper")
	shipper := newShipper(ctx)
	shipper.cl = ts.Client()
	singletonShipper = shipper
	defer shipper.Close()

	schedule := []time.Duration{0} // 0, 1, 1, 0, 0, 2, 2, 0, 1, 0, 2, 0}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		actual := model.Experiment{State: model.CompletedState}
		for id, delay := range schedule {
			time.Sleep(10 * delay * time.Millisecond)
			actual.ID = id
			t.Logf("reporting %d", actual.ID)
			require.NoError(t, ReportExperimentStateChanged(ctx, actual))
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		actual := map[int]int{} // Received IDs to count of hits.
		for range schedule {
			select {
			case exp := <-received:
				actual[exp.ID]++
			case <-ctx.Done():
				t.Error("webhook exited early")
				return
			}
		}

		actualIDs := maps.Keys(actual)
		slices.Sort(actualIDs)
		for i := 0; i < len(actualIDs)-1; i++ {
			require.Equalf(t, actualIDs[i]+1, actualIDs[i+1], "missing an id: %+v", actualIDs)
			require.Equal(t, 3, actual[i], "should've received event 3 times")
			require.Equal(t, 3, actual[i+1], "should've received event 3 times")
		}
	}()
	wg.Wait()
}
