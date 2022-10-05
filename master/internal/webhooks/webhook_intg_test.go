//go:build integration
// +build integration

package webhooks

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
)

func TestWebhooks(t *testing.T) {
	ctx := context.Background()
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, pathToMigrations)

	t.Run("webhook retrieval should work", func(t *testing.T) {
		testWebhookFour.Triggers = testWebhookFourTriggers
		testWebhookFive.Triggers = testWebhookFiveTriggers
		expectedWebhookIds := []WebhookID{testWebhookFour.ID, testWebhookFive.ID}
		err := AddWebhook(ctx, &testWebhookFour)
		err = AddWebhook(ctx, &testWebhookFive)
		require.NoError(t, err, "failure creating webhooks")
		webhooks, err := GetWebhooks(ctx)
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
		err := AddWebhook(ctx, &testWebhookOne)
		require.NoError(t, err, "failed to create webhook")
	})

	t.Run("webhook creation with multiple triggers should work", func(t *testing.T) {
		testWebhookTwo.Triggers = testTriggersTwo
		err := AddWebhook(ctx, &testWebhookTwo)
		require.NoError(t, err, "failed to create webhook with multiple triggers")
		webhooks, err := GetWebhooks(ctx)
		createdWebhook := getWebhookById(webhooks, testWebhookTwo.ID)
		require.Equal(t, len(createdWebhook.Triggers), len(testTriggersTwo), "did not retriee correct number of triggers")
	})

	t.Run("Deleting a webhook should work", func(t *testing.T) {
		testWebhookThree.Triggers = testTriggersThree

		err := AddWebhook(ctx, &testWebhookThree)
		require.NoError(t, err, "failed to create webhook")

		err = DeleteWebhook(ctx, testWebhookThree.ID)
		require.NoError(t, err, "errored when deleting webhook")
	})

	t.Cleanup(func() { cleanUp(ctx, t) })
}

var (
	testWebhookOne = Webhook{
		ID:          1000,
		URL:         "http://testwebhook.com",
		WebhookType: WebhookTypeSlack,
	}
	testWebhookTwo = Webhook{
		ID:          2000,
		URL:         "http://testwebhooktwo.com",
		WebhookType: WebhookTypeDefault,
	}
	testWebhookThree = Webhook{
		ID:          3000,
		URL:         "http://testwebhookthree.com",
		WebhookType: WebhookTypeSlack,
	}
	testWebhookFour = Webhook{
		ID:          6000,
		URL:         "http://twebhook.com",
		WebhookType: WebhookTypeSlack,
	}
	testWebhookFive = Webhook{
		ID:          7000,
		URL:         "http://twebhooktwo.com",
		WebhookType: WebhookTypeDefault,
	}
	testWebhookFourTrigger = Trigger{
		ID:          6001,
		TriggerType: TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   6000,
	}
	testWebhookFiveTrigger = Trigger{
		ID:          7001,
		TriggerType: TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   7000,
	}
	testWebhookFourTriggers = []*Trigger{&testWebhookFourTrigger}
	testWebhookFiveTriggers = []*Trigger{&testWebhookFiveTrigger}
	testTriggerOne          = Trigger{
		ID:          1001,
		TriggerType: TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   1000,
	}
	testTriggersOne     = []*Trigger{&testTriggerOne}
	testTriggerTwoState = Trigger{
		ID:          2001,
		TriggerType: TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   2000,
	}
	testTriggerTwoMetric = Trigger{
		ID:          2002,
		TriggerType: TriggerTypeMetricThresholdExceeded,
		Condition: map[string]interface{}{
			"metricName":  "validation_accuracy",
			"metricValue": 0.95,
		},
		WebhookID: 2000,
	}
	testTriggersTwo  = []*Trigger{&testTriggerTwoState, &testTriggerTwoMetric}
	testTriggerThree = Trigger{
		ID:          3001,
		TriggerType: TriggerTypeStateChange,
		Condition:   map[string]interface{}{"state": "COMPLETED"},
		WebhookID:   3000,
	}
	testTriggersThree = []*Trigger{&testTriggerThree}
)

const (
	pathToMigrations = "file://../../static/migrations"
)

func cleanUp(ctx context.Context, t *testing.T) {
	err := DeleteWebhook(ctx, testWebhookOne.ID)
	err = DeleteWebhook(ctx, testWebhookTwo.ID)
	err = DeleteWebhook(ctx, testWebhookThree.ID)
	err = DeleteWebhook(ctx, testWebhookFour.ID)
	err = DeleteWebhook(ctx, testWebhookFive.ID)
	if err != nil {
		t.Logf("error cleaning up webhook: %v", err)
	}
}

func getWebhookIds(webhooks Webhooks) []WebhookID {
	webhookIds := []WebhookID{}
	for _, w := range webhooks {
		webhookIds = append(webhookIds, w.ID)
	}
	return webhookIds
}

func getWebhookById(webhooks Webhooks, webhookId WebhookID) Webhook {
	for _, w := range webhooks {
		if w.ID == webhookId {
			return w
		}
	}
	return Webhook{}
}
