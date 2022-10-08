//go:build integration

package webhooks_test

import (
	"context"
	"testing"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/webhooks"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestWebhooks(t *testing.T) {
	ctx := context.Background()
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, pathToMigrations)

	_, err := db.Bun().NewDelete().Model((*webhooks.Webhook)(nil)).Where("1 = 1").Exec(ctx)
	require.NoError(t, err)

	t.Run("webhook retrieval should work", func(t *testing.T) {
		testWebhookFour.Triggers = testWebhookFourTriggers
		testWebhookFive.Triggers = testWebhookFiveTriggers
		expectedWebhookIds := []webhooks.WebhookID{testWebhookFour.ID, testWebhookFive.ID}
		err := webhooks.AddWebhook(ctx, &testWebhookFour)
		err = webhooks.AddWebhook(ctx, &testWebhookFive)
		require.NoError(t, err, "failure creating webhooks")
		webhooks, err := webhooks.GetWebhooks(ctx)
		webhookFourResponse := getWebhookById(webhooks, testWebhookFour.ID)
		require.NoError(t, err, "unable to get webhooks")
		require.Equal(t, len(webhooks), 2, "did not retrieve two webhooks")
		require.Equal(t, getWebhookIds(webhooks), expectedWebhookIds, "get request returned incorrect webhook Ids")
		require.Equal(t, len(webhooks), 2, "did not retrieve two webhooks")
		require.Equal(t, webhookFourResponse.URL, testWebhookFour.URL, "returned webhook url did not match")
		require.Equal(t, webhookFourResponse.WebhookType, testWebhookFour.WebhookType, "returned webhook type did not match")
	})

	t.Run("webhook creation should work", func(t *testing.T) {
		testWebhookOne.Triggers = testTriggersOne
		err := webhooks.AddWebhook(ctx, &testWebhookOne)
		require.NoError(t, err, "failed to create webhook")
	})

	t.Run("webhook creation with multiple triggers should work", func(t *testing.T) {
		testWebhookTwo.Triggers = testTriggersTwo
		err := webhooks.AddWebhook(ctx, &testWebhookTwo)
		require.NoError(t, err, "failed to create webhook with multiple triggers")
		webhooks, err := webhooks.GetWebhooks(ctx)
		createdWebhook := getWebhookById(webhooks, testWebhookTwo.ID)
		require.Equal(t, len(createdWebhook.Triggers), len(testTriggersTwo), "did not retriee correct number of triggers")
	})

	t.Run("Deleting a webhook should work", func(t *testing.T) {
		testWebhookThree.Triggers = testTriggersThree

		err := webhooks.AddWebhook(ctx, &testWebhookThree)
		require.NoError(t, err, "failed to create webhook")

		err = webhooks.DeleteWebhook(ctx, testWebhookThree.ID)
		require.NoError(t, err, "errored when deleting webhook")
	})

	t.Cleanup(func() { cleanUp(ctx, t) })
}

func TestReportExperimentStateChanged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, pathToMigrations)

	_, err := db.Bun().NewDelete().Model((*webhooks.Webhook)(nil)).Where("1 = 1").Exec(ctx)
	require.NoError(t, err)

	webhooks.Init(ctx)
	webhooks.Deinit() // We don't care to send, so just remove the events.

	t.Run("no triggers for event type", func(t *testing.T) {
		startCount, err := webhooks.CountEvents(ctx)
		require.NoError(t, err)

		require.NoError(t, webhooks.AddWebhook(ctx, mockWebhook()))
		require.NoError(t, webhooks.ReportExperimentStateChanged(ctx, model.Experiment{
			State: model.CanceledState,
		}))

		endCount, err := webhooks.CountEvents(ctx)
		require.NoError(t, err)
		require.Equal(t, startCount, endCount)
	})

	t.Run("no match triggers for event type", func(t *testing.T) {
		startCount, err := webhooks.CountEvents(ctx)
		require.NoError(t, err)

		w := mockWebhook()
		w.Triggers = append(w.Triggers, &webhooks.Trigger{
			TriggerType: webhooks.TriggerTypeStateChange,
			Condition:   map[string]interface{}{"state": model.CompletedState},
		})
		require.NoError(t, webhooks.AddWebhook(ctx, w))
		require.NoError(t, webhooks.ReportExperimentStateChanged(ctx, model.Experiment{
			State: model.CanceledState,
		}))

		endCount, err := webhooks.CountEvents(ctx)
		require.NoError(t, err)
		require.Equal(t, startCount, endCount)
	})

	_, err = db.Bun().NewDelete().Model((*webhooks.Webhook)(nil)).Where("1 = 1").Exec(ctx)
	require.NoError(t, err)

	t.Run("one trigger for event type", func(t *testing.T) {
		startCount, err := webhooks.CountEvents(ctx)
		require.NoError(t, err)

		w := mockWebhook()
		w.Triggers = append(w.Triggers, &webhooks.Trigger{
			TriggerType: webhooks.TriggerTypeStateChange,
			Condition:   map[string]interface{}{"state": model.CompletedState},
		})
		require.NoError(t, webhooks.AddWebhook(ctx, w))
		require.NoError(t, webhooks.ReportExperimentStateChanged(ctx, model.Experiment{
			State: model.CompletedState,
		}))

		endCount, err := webhooks.CountEvents(ctx)
		require.NoError(t, err)
		require.Equal(t, startCount+1, endCount)
	})

	_, err = db.Bun().NewDelete().Model((*webhooks.Webhook)(nil)).Where("1 = 1").Exec(ctx)
	require.NoError(t, err)

	t.Run("many triggers for event type", func(t *testing.T) {
		startCount, err := webhooks.CountEvents(ctx)
		require.NoError(t, err)

		w := mockWebhook()
		n := 10
		for i := 0; i < n; i++ {
			w.Triggers = append(w.Triggers, &webhooks.Trigger{
				TriggerType: webhooks.TriggerTypeStateChange,
				Condition:   map[string]interface{}{"state": model.CompletedState},
			})
		}
		require.NoError(t, webhooks.AddWebhook(ctx, w))
		require.NoError(t, webhooks.ReportExperimentStateChanged(ctx, model.Experiment{
			State: model.CompletedState,
		}))

		endCount, err := webhooks.CountEvents(ctx)
		require.NoError(t, err)
		require.Equal(t, startCount+n, endCount)
	})
}

var (
	testWebhookOne = webhooks.Webhook{
		ID:          1000,
		URL:         "http://testwebhook.com",
		WebhookType: webhooks.WebhookTypeSlack,
	}
	testWebhookTwo = webhooks.Webhook{
		ID:          2000,
		URL:         "http://testwebhooktwo.com",
		WebhookType: webhooks.WebhookTypeDefault,
	}
	testWebhookThree = webhooks.Webhook{
		ID:          3000,
		URL:         "http://testwebhookthree.com",
		WebhookType: webhooks.WebhookTypeSlack,
	}
	testWebhookFour = webhooks.Webhook{
		ID:          6000,
		URL:         "http://twebhook.com",
		WebhookType: webhooks.WebhookTypeSlack,
	}
	testWebhookFive = webhooks.Webhook{
		ID:          7000,
		URL:         "http://twebhooktwo.com",
		WebhookType: webhooks.WebhookTypeDefault,
	}
	testWebhookFourTrigger = webhooks.Trigger{
		ID:          6001,
		TriggerType: webhooks.TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   6000,
	}
	testWebhookFiveTrigger = webhooks.Trigger{
		ID:          7001,
		TriggerType: webhooks.TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   7000,
	}
	testWebhookFourTriggers = []*webhooks.Trigger{&testWebhookFourTrigger}
	testWebhookFiveTriggers = []*webhooks.Trigger{&testWebhookFiveTrigger}
	testTriggerOne          = webhooks.Trigger{
		ID:          1001,
		TriggerType: webhooks.TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   1000,
	}
	testTriggersOne     = []*webhooks.Trigger{&testTriggerOne}
	testTriggerTwoState = webhooks.Trigger{
		ID:          2001,
		TriggerType: webhooks.TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   2000,
	}
	testTriggerTwoMetric = webhooks.Trigger{
		ID:          2002,
		TriggerType: webhooks.TriggerTypeMetricThresholdExceeded,
		Condition: map[string]interface{}{
			"metricName":  "validation_accuracy",
			"metricValue": 0.95,
		},
		WebhookID: 2000,
	}
	testTriggersTwo  = []*webhooks.Trigger{&testTriggerTwoState, &testTriggerTwoMetric}
	testTriggerThree = webhooks.Trigger{
		ID:          3001,
		TriggerType: webhooks.TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   3000,
	}
	testTriggersThree = []*webhooks.Trigger{&testTriggerThree}
)

const (
	pathToMigrations = "file://../../static/migrations"
)

func cleanUp(ctx context.Context, t *testing.T) {
	err := webhooks.DeleteWebhook(ctx, testWebhookOne.ID)
	err = webhooks.DeleteWebhook(ctx, testWebhookTwo.ID)
	err = webhooks.DeleteWebhook(ctx, testWebhookThree.ID)
	err = webhooks.DeleteWebhook(ctx, testWebhookFour.ID)
	err = webhooks.DeleteWebhook(ctx, testWebhookFive.ID)
	if err != nil {
		t.Logf("error cleaning up webhook: %v", err)
	}
}

func getWebhookIds(ws webhooks.Webhooks) []webhooks.WebhookID {
	ids := []webhooks.WebhookID{}
	for _, w := range ws {
		ids = append(ids, w.ID)
	}
	return ids
}

func getWebhookById(ws webhooks.Webhooks, id webhooks.WebhookID) webhooks.Webhook {
	for _, w := range ws {
		if w.ID == id {
			return w
		}
	}
	return webhooks.Webhook{}
}

func mockWebhook() *webhooks.Webhook {
	return &webhooks.Webhook{
		URL:         "localhost:8080",
		WebhookType: webhooks.WebhookTypeDefault,
	}
}
