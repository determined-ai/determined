//go:build integration

package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	testURL = "localhost:8181"
)

func TestShipper(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	clearWebhooksTables(ctx, t)

	t.Run("payload signature generation should work", func(t *testing.T) {
		ts := int64(1666890081)
		url := "http://localhost:8080"
		name := expconf.Name{RawString: ptrs.Ptr("test-name")}
		e := ExperimentPayload{
			ID:            1,
			State:         "COMPLETED",
			Name:          name,
			Duration:      1,
			ResourcePool:  "default",
			SlotsPerTrial: 1,
			WorkspaceName: "workspace",
			ProjectName:   "project",
		}
		b, _ := json.Marshal(e)
		req, _ := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(b))
		key := []byte("testsigningkey")
		signedPayload := generateSignedPayload(req, ts, key)

		require.Equal(t, signedPayload,
			"899cc042278415da7d91605ffefe81376d64a4d842aa5663cd614f497f88910f")
	})

	t.Log("setup test webhook receiver")
	received := make(chan EventPayload, 100)
	mux := http.NewServeMux()
	mux.HandleFunc("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e EventPayload
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			t.Logf("error unmarshaling webhook body: %v", err)
			t.FailNow()
			return
		}
		received <- e
	}))
	go func() {
		defer cancel()
		if err := http.ListenAndServe(testURL, mux); err != nil { //nolint: gosec // This is a test.
			t.Logf("http receiver failed: %v", err)
		}
	}()

	t.Log("setup a few test webhooks")
	// One with two triggers so it fires twice.
	require.NoError(t, AddWebhook(ctx, &Webhook{
		URL: "http://" + testURL,
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
		URL: "http://" + testURL,
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
					"state": model.CanceledState, // One that shouldn't fire, for fun.
				},
			},
		},
		WebhookType: WebhookTypeDefault,
	}))

	t.Log("build shipper")
	singletonShipper = newShipper() // set the singleton so reports can find it.
	defer func() {
		t.Log("closing shipper")
		// Last event may get rolled back because the shipper is closed too quickly - that's OK.
		singletonShipper.Close()
	}()

	schedule := []int{0, 0, 1, 1, 0, 0, 2, 2, 0, 1, 0, 2, 0, 1, 0, 2, 0, 1, 0}
	var progress atomic.Int64

	// Because the shipper singleton is not thread-safe, we cannot reset it later without this lock.
	var shipperInitLock sync.Mutex

	expected := map[int]int{} // Sent IDs to count of expected hits, access protected by waitgroup.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var config expconf.ExperimentConfig
		config = schemas.WithDefaults(config)
		for id, delay := range schedule {
			time.Sleep(scheduledWaitToDuration(delay))
			expected[id] = 3 // 3 sends, one for each trigger.
			shipperInitLock.Lock()
			err := ReportExperimentStateChanged(ctx, model.Experiment{
				ID:    id,
				State: model.CompletedState,
			}, config)
			shipperInitLock.Unlock()
			require.NoError(t, err)
			progress.Store(int64(id))
		}
	}()

	actual := map[int]map[uuid.UUID]int{} // Received experiment IDs to event ID to count of hits.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for uniqRecv := 0; uniqRecv < len(schedule)*3; {
			select {
			case event := <-received:
				if actual[event.Data.Experiment.ID] == nil {
					actual[event.Data.Experiment.ID] = map[uuid.UUID]int{}
				}
				if actual[event.Data.Experiment.ID][event.ID] == 0 {
					uniqRecv++
				}
				actual[event.Data.Experiment.ID][event.ID]++
			case <-ctx.Done():
				t.Error("webhook exited early")
				return
			}
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// Test fault tolerance by closing the shipper halfway through, recreating it, and ensuring all
	// events make it at least once anyway.
	totalWait := 0
	for _, wait := range schedule {
		totalWait += wait
	}
	time.Sleep(scheduledWaitToDuration(totalWait) / 2)
	t.Logf("chaosing shipper with %d/%d events sent", progress.Load(), len(schedule))
	singletonShipper.Close()
	t.Log("recreating shipper")
	shipperInitLock.Lock()
	singletonShipper = newShipper()
	shipperInitLock.Unlock()

	select {
	case <-done:
		t.Log("waitgroup closed, checking results")
		require.ElementsMatch(t, maps.Keys(expected), maps.Keys(actual), "missing events for exps")
		for expID, events := range actual {
			require.Equalf(t, 3, len(events), "missing events for exp %d", expID)
			for _, sends := range events {
				require.GreaterOrEqual(t, sends, 1, "event was not sent at least once")
				require.LessOrEqual(t, sends, 3, "event was not sent an excessive number of times")
			}
		}
	case <-time.After(10 * time.Second):
		t.Errorf("did not receive all events in time")
	case <-ctx.Done():
		t.Errorf("exited: %s", ctx.Err())
	}
}

func scheduledWaitToDuration(factor int) time.Duration {
	return 10 * time.Duration(factor) * time.Millisecond
}
