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
)

const pathToMigrations = "file://../../static/migrations"

// func TestShipper(t *testing.T) {
// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	pgDB := db.MustResolveTestPostgres(t)
// 	db.MustMigrateTestPostgres(t, pgDB, pathToMigrations)

// 	_, err := db.Bun().NewDelete().Model((*Webhook)(nil)).Where("1 = 1").Exec(ctx)
// 	require.NoError(t, err)

// 	var actual *model.Experiment
// 	var postsLock sync.Mutex
// 	received := make(chan struct{})
// 	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		var e model.Experiment
// 		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
// 			t.Logf("error decoding webhook: %v", err)
// 			t.FailNow()
// 			return
// 		}
// 		t.Logf("received event for experiment: %v", e.ID)

// 		postsLock.Lock()
// 		defer postsLock.Unlock()
// 		actual = &e
// 		received <- struct{}{}
// 	}))
// 	url = ts.URL
// 	defer ts.Close()

// 	Init(ctx)
// 	defer Deinit()

// 	t.Run("no triggers for event type", func(t *testing.T) {
// 		startCount, err := webhooks.CountEvents(ctx)
// 		require.NoError(t, err)

// 		require.NoError(t, webhooks.AddWebhook(ctx, mockWebhook()))
// 		require.NoError(t, webhooks.ReportExperimentStateChanged(ctx, model.Experiment{
// 			State: model.CanceledState,
// 		}))

// 		endCount, err := webhooks.CountEvents(ctx)
// 		require.NoError(t, err)
// 		require.Equal(t, startCount, endCount)
// 	})

// 	t.Run("no match triggers for event type", func(t *testing.T) {
// 		startCount, err := webhooks.CountEvents(ctx)
// 		require.NoError(t, err)

// 		w := mockWebhook()
// 		w.Triggers = append(w.Triggers, &webhooks.Trigger{
// 			TriggerType: webhooks.TriggerTypeStateChange,
// 			Condition:   map[string]interface{}{"state": model.CompletedState},
// 		})
// 		require.NoError(t, webhooks.AddWebhook(ctx, w))
// 		require.NoError(t, webhooks.ReportExperimentStateChanged(ctx, model.Experiment{
// 			State: model.CanceledState,
// 		}))

// 		endCount, err := webhooks.CountEvents(ctx)
// 		require.NoError(t, err)
// 		require.Equal(t, startCount, endCount)
// 	})

// 	_, err = db.Bun().NewDelete().Model((*webhooks.Webhook)(nil)).Where("1 = 1").Exec(ctx)
// 	require.NoError(t, err)

// 	t.Run("one trigger for event type", func(t *testing.T) {
// 		startCount, err := webhooks.CountEvents(ctx)
// 		require.NoError(t, err)

// 		w := mockWebhook()
// 		w.Triggers = append(w.Triggers, &webhooks.Trigger{
// 			TriggerType: webhooks.TriggerTypeStateChange,
// 			Condition:   map[string]interface{}{"state": model.CompletedState},
// 		})
// 		require.NoError(t, webhooks.AddWebhook(ctx, w))
// 		require.NoError(t, webhooks.ReportExperimentStateChanged(ctx, model.Experiment{
// 			State: model.CompletedState,
// 		}))

// 		endCount, err := webhooks.CountEvents(ctx)
// 		require.NoError(t, err)
// 		require.Equal(t, startCount+1, endCount)
// 	})

// 	_, err = db.Bun().NewDelete().Model((*webhooks.Webhook)(nil)).Where("1 = 1").Exec(ctx)
// 	require.NoError(t, err)

// 	t.Run("many triggers for event type", func(t *testing.T) {
// 		startCount, err := webhooks.CountEvents(ctx)
// 		require.NoError(t, err)

// 		w := mockWebhook()
// 		n := 10
// 		for i := 0; i < n; i++ {
// 			w.Triggers = append(w.Triggers, &webhooks.Trigger{
// 				TriggerType: webhooks.TriggerTypeStateChange,
// 				Condition:   map[string]interface{}{"state": model.CompletedState},
// 			})
// 		}
// 		require.NoError(t, webhooks.AddWebhook(ctx, w))
// 		require.NoError(t, webhooks.ReportExperimentStateChanged(ctx, model.Experiment{
// 			State: model.CompletedState,
// 		}))

// 		endCount, err := webhooks.CountEvents(ctx)
// 		require.NoError(t, err)
// 		require.Equal(t, startCount+n, endCount)
// 	})
// }

func TestSender(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, pathToMigrations)

	_, err := db.Bun().NewDelete().Model((*Webhook)(nil)).Where("1=1").Exec(ctx)
	require.NoError(t, err)
	_, err = db.Bun().NewDelete().Model((*Event)(nil)).Where("1=1").Exec(ctx)
	require.NoError(t, err)

	var actual *model.Experiment
	var postsLock sync.Mutex
	received := make(chan struct{})
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var e model.Experiment
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			t.Logf("error decoding webhook: %v", err)
			t.FailNow()
			return
		}
		t.Logf("received event for experiment: %v", e.ID)

		postsLock.Lock()
		defer postsLock.Unlock()
		actual = &e
		received <- struct{}{}
	}))
	url = ts.URL
	defer ts.Close()

	Init(ctx)
	defer Deinit()
	singletonShipper.cl = ts.Client()

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

	expected := model.Experiment{ID: 99, State: model.CompletedState}
	require.NoError(t, ReportExperimentStateChanged(ctx, expected))

	ctx, cancel = context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	select {
	case <-received:
	case <-ctx.Done():
		t.Log("webhook receive timed out")
		t.FailNow()
	}

	expectedBytes, err := json.Marshal(expected)
	require.NoError(t, err)

	actualBytes, err := json.Marshal(actual)
	require.NoError(t, err)

	require.JSONEq(t, string(expectedBytes), string(actualBytes))

	n, err := CountEvents(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, n, "there should be no events left")
}

func mockWebhook() *Webhook {
	return &Webhook{
		URL:         "localhost:8080",
		WebhookType: WebhookTypeDefault,
	}
}
