//go:build integration

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

func TestWebhooks(t *testing.T) {
	ctx := context.Background()
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	clearWebhooksTables(ctx, t)

	t.Run("webhook retrieval should work", func(t *testing.T) {
		testWebhookFour.Triggers = testWebhookFourTriggers
		testWebhookFive.Triggers = testWebhookFiveTriggers
		expectedWebhookIds := []WebhookID{testWebhookFour.ID, testWebhookFive.ID}
		err := AddWebhook(ctx, &testWebhookFour)
		require.NoError(t, err)
		err = AddWebhook(ctx, &testWebhookFive)
		require.NoError(t, err, "failure creating webhooks")
		webhooks, err := GetWebhooks(ctx)
		webhookFourResponse := getWebhookByID(webhooks, testWebhookFour.ID)
		require.NoError(t, err, "unable to get webhooks")
		require.Equal(t, len(webhooks), 2, "did not retrieve two webhooks")
		require.Equal(t, getWebhookIds(webhooks), expectedWebhookIds,
			"get request returned incorrect webhook Ids")
		require.Equal(t, len(webhooks), 2, "did not retrieve two webhooks")
		require.Equal(t, webhookFourResponse.URL, testWebhookFour.URL,
			"returned webhook url did not match")
		require.Equal(t, webhookFourResponse.WebhookType, testWebhookFour.WebhookType,
			"returned webhook type did not match")
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
		require.NoError(t, err)
		createdWebhook := getWebhookByID(webhooks, testWebhookTwo.ID)
		require.Equal(t, len(createdWebhook.Triggers), len(testTriggersTwo),
			"did not retriee correct number of triggers")
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

func TestReportLogPatternAction(t *testing.T) {
	ctx := context.Background()

	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	clearWebhooksTables(ctx, t)

	originalConfig := config.GetMasterConfig().Webhooks
	defer func() {
		config.GetMasterConfig().Webhooks = originalConfig
	}()

	for _, webhookType := range []WebhookType{WebhookTypeDefault, WebhookTypeSlack} {
		for _, baseURLIsSet := range []bool{true, false} {
			if baseURLIsSet {
				config.GetMasterConfig().Webhooks.BaseURL = "http://determined.ai"
			} else {
				config.GetMasterConfig().Webhooks.BaseURL = ""
			}

			t.Run(fmt.Sprintf(
				"webhookType=%v baseURLIsSet=%v", webhookType, baseURLIsSet), func(t *testing.T) {
				user := db.RequireMockUser(t, pgDB)
				exp := db.RequireMockExperiment(t, pgDB, user)
				trial, task := db.RequireMockTrial(t, pgDB, exp)

				expected := EventPayload{
					Type: TriggerTypeLogPatternPolicy,
					Condition: Condition{
						LogPatternPolicyRegex: "regexa",
					},
					Data: EventData{
						LogPatternPolicy: &LogPatternPolicyPayload{
							TaskID:        task.TaskID,
							NodeName:      "nodeA",
							TriggeringLog: "trigA",
						},
					},
				}

				uuidURL := uuid.New().String()
				require.NoError(t, ReportLogPatternAction(
					ctx,
					task.TaskID,
					expected.Data.LogPatternPolicy.NodeName,
					expected.Condition.LogPatternPolicyRegex,
					expected.Data.LogPatternPolicy.TriggeringLog,
					uuidURL,
					webhookType))

				var actualEvent Event
				require.NoError(t, db.Bun().NewSelect().Model(&actualEvent).
					Where("url = ?", uuidURL).
					Scan(ctx, &actualEvent))

				if webhookType == WebhookTypeDefault {
					var actual EventPayload
					require.NoError(t, json.Unmarshal(actualEvent.Payload, &actual))

					actual.ID = expected.ID
					actual.Timestamp = expected.Timestamp
					require.Equal(t, expected, actual)
					return
				}

				var actual SlackMessageBody
				require.NoError(t, json.Unmarshal(actualEvent.Payload, &actual))

				msg := fmt.Sprintf(
					"Experiment ID `%d`, Trial ID `%d`, running on node `nodeA`, reported a log\n",
					trial.ExperimentID, trial.ID) +
					"```trigA```\n" +
					"This log matched the regex\n" +
					"```regexa```\n"

				path := fmt.Sprintf("/det/experiments/%d/trials/%d/logs", trial.ExperimentID, trial.ID)
				if baseURLIsSet {
					msg += fmt.Sprintf("<http://determined.ai%s | View full logs here>", path)
				} else {
					msg += fmt.Sprintf("View full logs at %s", path)
				}

				require.Equal(t, SlackMessageBody{
					Blocks: []SlackBlock{
						{
							Type: "section",
							Text: SlackField{
								Type: "mrkdwn",
								Text: msg,
							},
						},
					},
				}, actual)
			})
		}
	}
}

func TestReportExperimentStateChanged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	clearWebhooksTables(ctx, t)

	singletonShipper = &shipper{wake: make(chan<- struct{})} // mock shipper

	var config expconf.ExperimentConfig
	config = schemas.WithDefaults(config)

	t.Run("no triggers for event type", func(t *testing.T) {
		w := mockWebhook()
		require.NoError(t, AddWebhook(ctx, w))
		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			State: model.CanceledState,
		}, config))

		require.Equal(t, 0, countEventsForURL(ctx, t, w.URL))
	})

	t.Run("no match triggers for event type", func(t *testing.T) {
		w := mockWebhook()
		w.Triggers = append(w.Triggers, &Trigger{
			TriggerType: TriggerTypeStateChange,
			Condition:   map[string]interface{}{"state": model.CompletedState},
		})
		require.NoError(t, AddWebhook(ctx, w))
		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			State: model.CanceledState,
		}, config))

		require.Equal(t, 0, countEventsForURL(ctx, t, w.URL))
	})

	clearWebhooksTables(ctx, t)

	t.Run("one trigger for event type", func(t *testing.T) {
		w := mockWebhook()
		w.Triggers = append(w.Triggers, &Trigger{
			TriggerType: TriggerTypeStateChange,
			Condition:   map[string]interface{}{"state": model.CompletedState},
		})
		require.NoError(t, AddWebhook(ctx, w))
		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			State: model.CompletedState,
		}, config))

		require.Equal(t, 1, countEventsForURL(ctx, t, w.URL))
	})

	clearWebhooksTables(ctx, t)

	t.Run("many triggers for event type", func(t *testing.T) {
		w := mockWebhook()
		n := 10
		for i := 0; i < n; i++ {
			w.Triggers = append(w.Triggers, &Trigger{
				TriggerType: TriggerTypeStateChange,
				Condition:   map[string]interface{}{"state": model.CompletedState},
			})
		}
		require.NoError(t, AddWebhook(ctx, w))
		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			State: model.CompletedState,
		}, config))

		require.Equal(t, n, countEventsForURL(ctx, t, w.URL))
	})
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

func cleanUp(ctx context.Context, t *testing.T) {
	require.NoError(t, DeleteWebhook(ctx, testWebhookOne.ID))
	require.NoError(t, DeleteWebhook(ctx, testWebhookTwo.ID))
	require.NoError(t, DeleteWebhook(ctx, testWebhookThree.ID))
	require.NoError(t, DeleteWebhook(ctx, testWebhookFour.ID))
	require.NoError(t, DeleteWebhook(ctx, testWebhookFive.ID))
}

func getWebhookIds(ws Webhooks) []WebhookID {
	ids := []WebhookID{}
	for _, w := range ws {
		ids = append(ids, w.ID)
	}
	return ids
}

func getWebhookByID(ws Webhooks, id WebhookID) Webhook {
	for _, w := range ws {
		if w.ID == id {
			return w
		}
	}
	return Webhook{}
}

func mockWebhook() *Webhook {
	return &Webhook{
		URL:         uuid.New().String(),
		WebhookType: WebhookTypeDefault,
	}
}

func TestDequeueEvents(t *testing.T) {
	ctx := context.Background()
	pgDB := db.MustResolveTestPostgres(t)
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	clearWebhooksTables(ctx, t)

	singletonShipper = &shipper{wake: make(chan<- struct{})} // mock shipper

	var config expconf.ExperimentConfig
	config = schemas.WithDefaults(config)

	t.Log("add a test webhook with one trigger")
	require.NoError(t, AddWebhook(ctx, &Webhook{
		URL: "localhost:8181",
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

	t.Run("dequeueing and consuming a event should work", func(t *testing.T) {
		exp := model.Experiment{
			State: model.CompletedState,
		}
		require.NoError(t, ReportExperimentStateChanged(ctx, exp, config))

		batch, err := dequeueEvents(ctx, maxEventBatchSize)
		require.NoError(t, batch.commit())
		require.NoError(t, err)
		require.Equal(t, 1, len(batch.events))
	})

	t.Run("dequeueing and consuming a full batch of events should work", func(t *testing.T) {
		for i := 0; i < maxEventBatchSize; i++ {
			exp := model.Experiment{ID: i, State: model.CompletedState}
			require.NoError(t, ReportExperimentStateChanged(ctx, exp, config))
		}

		batch, err := dequeueEvents(ctx, maxEventBatchSize)
		require.NoError(t, batch.commit())
		require.NoError(t, err)
		require.Equal(t, maxEventBatchSize, len(batch.events))
	})

	t.Run("rolling back an event should work, and it should be reconsumed", func(t *testing.T) {
		exp := model.Experiment{State: model.CompletedState}
		require.NoError(t, ReportExperimentStateChanged(ctx, exp, config))

		batch, err := dequeueEvents(ctx, maxEventBatchSize)
		require.NoError(t, err)
		require.NoError(t, batch.rollback())

		batch, err = dequeueEvents(ctx, maxEventBatchSize)
		require.NoError(t, batch.commit())
		require.NoError(t, err)
		require.Equal(t, 1, len(batch.events))
	})
}

func clearWebhooksTables(ctx context.Context, t *testing.T) {
	t.Log("clear webhooks db")
	_, err := db.Bun().NewDelete().Model((*Webhook)(nil)).Where("true").Exec(ctx)
	require.NoError(t, err)
	_, err = db.Bun().NewDelete().Model((*Event)(nil)).Where("true").Exec(ctx)
	require.NoError(t, err)
}

func countEventsForURL(ctx context.Context, t *testing.T, url string) int {
	c, err := db.Bun().NewSelect().Model((*Event)(nil)).
		Where("url = ?", url).
		Count(ctx)
	require.NoError(t, err)

	return c
}
