//go:build integration

package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/google/uuid"

	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/etc"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/master/pkg/schemas"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
)

const (
	webhookName1 = "test-webhook-name1"
	webhookName2 = "test-webhook-name2"
	webhookName3 = "test-webhook-name3"
)

var pgDB *db.PgDB

func TestMain(m *testing.M) {
	var err error
	pgDB, _, err = db.ResolveTestPostgres()
	if err != nil {
		log.Panicln(err)
	}

	err = db.MigrateTestPostgres(pgDB, "file://../../static/migrations", "up")
	if err != nil {
		log.Panicln(err)
	}

	err = etc.SetRootPath("../../static/srv")
	if err != nil {
		log.Panicln(err)
	}

	manager, err := New(context.TODO())
	if err != nil {
		panic(err)
	}
	SetDefault(manager)

	os.Exit(m.Run())
}

func TestWebhooks(t *testing.T) {
	ctx := context.Background()
	clearWebhooksTables(ctx, t)

	t.Run("webhook retrieval should work", func(t *testing.T) {
		testWebhookFour.Triggers = testWebhookFourTriggers
		testWebhookFive.Triggers = testWebhookFiveTriggers
		expectedWebhookIds := []WebhookID{testWebhookFour.ID, testWebhookFive.ID}
		err := AddWebhook(ctx, &testWebhookFour)
		require.NoError(t, err)
		err = AddWebhook(ctx, &testWebhookFive)
		require.NoError(t, err, "failure creating webhooks")
		webhooks, err := getWebhooks(ctx, nil)
		webhookFourResponse := getWebhookByID(webhooks, testWebhookFour.ID)
		require.NoError(t, err, "unable to get webhooks")
		require.Len(t, webhooks, 2, "did not retrieve two webhooks")
		require.Equal(t, expectedWebhookIds, getWebhookIds(webhooks),
			"get request returned incorrect webhook Ids")
		require.Len(t, webhooks, 2, "did not retrieve two webhooks")
		require.Equal(t, webhookFourResponse.URL, testWebhookFour.URL,
			"returned webhook url did not match")
		require.Equal(t, webhookFourResponse.WebhookType, testWebhookFour.WebhookType,
			"returned webhook type did not match")
	})

	t.Run("webhook creation should work", func(t *testing.T) {
		workspaceID := int32(1)
		testWebhookOne.Triggers = testTriggersOne
		testWebhookOne.Mode = WebhookModeSpecific
		testWebhookOne.Name = "test-name"
		testWebhookOne.WorkspaceID = &workspaceID
		err := AddWebhook(ctx, &testWebhookOne)
		require.NoError(t, err, "failed to create webhook")
		webhooks, err := getWebhooks(ctx, &[]int32{workspaceID})
		require.NoError(t, err, "unable to get webhooks")
		webhookOneResponse := getWebhookByID(webhooks, testWebhookOne.ID)
		require.Equal(t, "test-name", webhookOneResponse.Name)
		require.Equal(t, workspaceID, *webhookOneResponse.WorkspaceID)
		require.Equal(t, WebhookModeSpecific, testWebhookOne.Mode)
	})

	t.Run("webhook creation with multiple triggers should work", func(t *testing.T) {
		testWebhookTwo.Triggers = testTriggersTwo
		err := AddWebhook(ctx, &testWebhookTwo)
		require.NoError(t, err, "failed to create webhook with multiple triggers")
		webhooks, err := getWebhooks(ctx, nil)
		require.NoError(t, err)
		createdWebhook := getWebhookByID(webhooks, testWebhookTwo.ID)
		require.Len(t, createdWebhook.Triggers, len(testTriggersTwo),
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

func TestWebhookScanLogs(t *testing.T) {
	ctx := context.Background()
	clearWebhooksTables(ctx, t)

	manager, err := New(ctx)
	require.NoError(t, err)

	r0 := uuid.New().String()
	r1 := uuid.New().String()
	r2 := uuid.New().String()

	workspaceID, _ := db.RequireMockWorkspaceID(t, pgDB, "")

	w0 := &Webhook{
		WebhookType: WebhookTypeDefault,
		URL:         uuid.New().String(),
		Triggers: Triggers{
			{TriggerType: TriggerTypeTaskLog, Condition: map[string]any{"regex": r0}},
		},
		Mode: WebhookModeWorkspace,
	}
	w1 := &Webhook{
		WebhookType: WebhookTypeDefault,
		URL:         uuid.New().String(),
		Triggers: Triggers{
			{TriggerType: TriggerTypeTaskLog, Condition: map[string]any{"regex": r0}},
			{TriggerType: TriggerTypeTaskLog, Condition: map[string]any{"regex": r1}},
		},
		Mode: WebhookModeWorkspace,
	}
	w2 := &Webhook{
		WebhookType: WebhookTypeDefault,
		URL:         uuid.New().String(),
		Triggers: Triggers{
			{TriggerType: TriggerTypeTaskLog, Condition: map[string]any{"regex": r2}},
		},
		Mode: WebhookModeWorkspace,
	}
	w3 := &Webhook{
		WebhookType: WebhookTypeDefault,
		URL:         uuid.New().String(),
		Triggers: Triggers{
			{TriggerType: TriggerTypeTaskLog, Condition: map[string]any{"regex": r2}},
		},
		Name: webhookName1,
		Mode: WebhookModeSpecific,
	}
	w4 := &Webhook{
		WebhookType: WebhookTypeDefault,
		URL:         uuid.New().String(),
		Triggers: Triggers{
			{TriggerType: TriggerTypeTaskLog, Condition: map[string]any{"regex": r2}},
		},
		Name:        webhookName2,
		Mode:        WebhookModeSpecific,
		WorkspaceID: ptrs.Ptr(int32(workspaceID)),
	}
	w5 := &Webhook{
		WebhookType: WebhookTypeDefault,
		URL:         uuid.New().String(),
		Triggers: Triggers{
			{TriggerType: TriggerTypeTaskLog, Condition: map[string]any{"regex": r2}},
		},
		Mode:        WebhookModeWorkspace,
		WorkspaceID: ptrs.Ptr(int32(workspaceID)),
	}

	require.NoError(t, manager.addWebhook(ctx, w0))
	require.NoError(t, manager.addWebhook(ctx, w1))
	require.NoError(t, manager.addWebhook(ctx, w2))
	require.NoError(t, manager.addWebhook(ctx, w3))
	require.NoError(t, manager.addWebhook(ctx, w4))
	require.NoError(t, manager.addWebhook(ctx, w5))

	for _, shouldBounce := range []bool{false, true} { //nolint: dupl
		clearWebhooksEvent(ctx, t)

		if shouldBounce {
			manager, err = New(ctx)
			require.NoError(t, err)
		}

		user := db.RequireMockUser(t, pgDB)
		exp := db.RequireMockExperiment(t, pgDB, user)
		_, task := db.RequireMockTrial(t, pgDB, exp)

		logs := []*model.TaskLog{
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r0},
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r0},
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r1},
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r2},
		}
		require.NoError(t, manager.scanLogs(ctx, logs, 0, nil))

		require.Equal(t, 1, countEventsForURL(ctx, t, w0.URL))
		require.Equal(t, 2, countEventsForURL(ctx, t, w1.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w2.URL))
		require.Zero(t, countEventsForURL(ctx, t, w3.URL))
		require.Zero(t, countEventsForURL(ctx, t, w4.URL))
		require.Zero(t, countEventsForURL(ctx, t, w5.URL))

		clearWebhooksEvent(ctx, t)

		require.NoError(t, manager.scanLogs(ctx, logs, model.AccessScopeID(workspaceID), nil))
		require.Equal(t, 1, countEventsForURL(ctx, t, w0.URL))
		require.Equal(t, 2, countEventsForURL(ctx, t, w1.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w2.URL))
		require.Zero(t, countEventsForURL(ctx, t, w3.URL))
		require.Zero(t, countEventsForURL(ctx, t, w4.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w5.URL))

		clearWebhooksEvent(ctx, t)

		exp = db.RequireMockExperimentParams(t, pgDB, user,
			db.MockExperimentParams{
				Integrations: &expconf.IntegrationsConfigV0{
					Webhooks: &expconf.WebhooksConfigV0{
						WebhookName: ptrs.Ptr([]string{webhookName1, webhookName2}),
					},
				},
			},
			db.DefaultProjectID)
		_, task = db.RequireMockTrial(t, pgDB, exp)

		logs = []*model.TaskLog{
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r2},
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r2},
		}
		require.NoError(t, manager.scanLogs(ctx, logs, 0, ptrs.Ptr(exp.ID)))

		require.Equal(t, 1, countEventsForURL(ctx, t, w2.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w3.URL))
		require.Zero(t, countEventsForURL(ctx, t, w4.URL))
		require.Zero(t, countEventsForURL(ctx, t, w5.URL))

		clearWebhooksEvent(ctx, t)

		require.NoError(t, manager.scanLogs(ctx, logs, model.AccessScopeID(workspaceID), ptrs.Ptr(exp.ID)))

		require.Equal(t, 1, countEventsForURL(ctx, t, w2.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w3.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w4.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w5.URL))

		clearWebhooksEvent(ctx, t)

		exp = db.RequireMockExperimentParams(t, pgDB, user,
			db.MockExperimentParams{
				Integrations: &expconf.IntegrationsConfigV0{
					Webhooks: &expconf.WebhooksConfigV0{
						Exclude: true,
					},
				},
			},
			db.DefaultProjectID)
		_, task = db.RequireMockTrial(t, pgDB, exp)

		logs = []*model.TaskLog{
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r2},
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r2},
		}

		require.NoError(t, manager.scanLogs(ctx, logs, model.AccessScopeID(workspaceID), ptrs.Ptr(exp.ID)))

		require.Zero(t, countEventsForURL(ctx, t, w0.URL))
		require.Zero(t, countEventsForURL(ctx, t, w1.URL))
		require.Zero(t, countEventsForURL(ctx, t, w2.URL))
		require.Zero(t, countEventsForURL(ctx, t, w3.URL))
		require.Zero(t, countEventsForURL(ctx, t, w4.URL))
		require.Zero(t, countEventsForURL(ctx, t, w5.URL))
	}

	require.NoError(t, manager.deleteWebhook(ctx, w1.ID))

	for _, shouldBounce := range []bool{false, true} { //nolint: dupl
		_, err = db.Bun().NewDelete().Model((*Event)(nil)).Where("true").Exec(ctx)
		require.NoError(t, err)

		if shouldBounce {
			manager, err = New(ctx)
			require.NoError(t, err)
		}

		user := db.RequireMockUser(t, pgDB)
		exp := db.RequireMockExperiment(t, pgDB, user)
		_, task := db.RequireMockTrial(t, pgDB, exp)

		logs := []*model.TaskLog{
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r0},
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r0},
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r1},
			{TaskID: string(task.TaskID), AgentID: ptrs.Ptr("test"), Log: r2},
		}
		require.NoError(t, manager.scanLogs(ctx, logs, 0, nil))

		require.Equal(t, 1, countEventsForURL(ctx, t, w0.URL))
		require.Zero(t, countEventsForURL(ctx, t, w1.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w2.URL))
	}
}

func TestGenerateTaskLogPayload(t *testing.T) {
	ctx := context.Background()

	originalConfig := config.GetMasterConfig().Webhooks
	defer func() {
		config.GetMasterConfig().Webhooks = originalConfig
	}()

	for _, webhookType := range []WebhookType{WebhookTypeDefault, WebhookTypeSlack} {
		for _, baseURLIsSet := range []bool{true, false} {
			for _, taskIsExp := range []bool{true, false} {
				if baseURLIsSet {
					config.GetMasterConfig().Webhooks.BaseURL = "http://determined.ai"
				} else {
					config.GetMasterConfig().Webhooks.BaseURL = ""
				}

				testGenerateTaskLogPayloadTest(ctx, t, webhookType, baseURLIsSet, taskIsExp)
			}
		}
	}
}

func testGenerateTaskLogPayloadTest(
	ctx context.Context, t *testing.T, webhookType WebhookType, baseURLIsSet bool, taskIsExp bool,
) {
	t.Run(fmt.Sprintf("webhookType=%v baseURLIsSet=%v taskIsExp=%v",
		webhookType, baseURLIsSet, taskIsExp), func(t *testing.T) {
		var task *model.Task
		var trial *model.Trial
		if taskIsExp {
			user := db.RequireMockUser(t, pgDB)
			exp := db.RequireMockExperiment(t, pgDB, user)

			trial, task = db.RequireMockTrial(t, pgDB, exp)
		} else {
			task = &model.Task{
				TaskID:    model.NewTaskID(),
				JobID:     nil,
				TaskType:  model.TaskTypeNotebook,
				StartTime: time.Now().UTC().Truncate(time.Millisecond),
			}
			require.NoError(t, db.AddTask(ctx, task))
		}

		payload, err := generateTaskLogPayload(
			ctx, task.TaskID, "nodeA", "regexa", "trigA", webhookType)
		require.NoError(t, err)

		if webhookType == WebhookTypeDefault {
			expected := EventPayload{
				Type: TriggerTypeTaskLog,
				Condition: Condition{
					Regex: "regexa",
				},
				Data: EventData{
					TaskLog: &TaskLogPayload{
						TaskID:        task.TaskID,
						NodeName:      "nodeA",
						TriggeringLog: "trigA",
					},
				},
			}

			var actual EventPayload
			require.NoError(t, json.Unmarshal(payload, &actual))

			actual.ID = expected.ID
			actual.Timestamp = expected.Timestamp
			require.Equal(t, expected, actual)
			return
		}

		var actual SlackMessageBody
		require.NoError(t, json.Unmarshal(payload, &actual))

		msg := ""
		if taskIsExp {
			msg = fmt.Sprintf(
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
		} else {
			msg = fmt.Sprintf(
				"Task ID `%s`, task type `%s`, running on node `nodeA`, reported a log\n",
				task.TaskID, model.TaskTypeNotebook) +
				"```trigA```\n" +
				"This log matched the regex\n" +
				"```regexa```\n"
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

func TestReportExperimentStateChanged(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	singletonShipper = &shipper{wake: make(chan<- struct{})} // mock shipper

	workspaceName := uuid.New().String()
	workspaceID, _ := db.RequireMockWorkspaceID(t, pgDB, workspaceName)
	projectID, _ := db.RequireMockProjectID(t, pgDB, workspaceID, false)

	var config expconf.ExperimentConfig
	config = schemas.WithDefaults(config)

	t.Run("no triggers for event type", func(t *testing.T) {
		w := mockWebhook()
		require.NoError(t, AddWebhook(ctx, w))
		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			ID:        0,
			ProjectID: projectID,
			State:     model.CanceledState,
		}, config))

		require.Zero(t, countEventsForURL(ctx, t, w.URL))
	})

	t.Run("no match triggers for event type", func(t *testing.T) {
		w := mockWebhook()
		w.Triggers = append(w.Triggers, &Trigger{
			TriggerType: TriggerTypeStateChange,
			Condition:   map[string]interface{}{"state": model.CompletedState},
		})
		require.NoError(t, AddWebhook(ctx, w))
		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			ID:        0,
			ProjectID: projectID,
			State:     model.CanceledState,
		}, config))

		require.Zero(t, countEventsForURL(ctx, t, w.URL))
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
			ID:        0,
			ProjectID: projectID,
			State:     model.CompletedState,
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
			ID:        0,
			ProjectID: projectID,
			State:     model.CompletedState,
		}, config))

		require.Equal(t, n, countEventsForURL(ctx, t, w.URL))
	})

	clearWebhooksTables(ctx, t)
	t.Run("webhook with mode specific", func(t *testing.T) {
		w1 := &Webhook{
			URL:         uuid.New().String(),
			WebhookType: WebhookTypeDefault,
			Mode:        WebhookModeSpecific,
			Triggers: Triggers{
				{
					TriggerType: TriggerTypeStateChange,
					Condition:   map[string]interface{}{"state": model.CompletedState},
				},
			},
			WorkspaceID: ptrs.Ptr(int32(model.DefaultWorkspaceID)),
			Name:        webhookName1,
		}
		w2 := &Webhook{
			URL:         uuid.New().String(),
			WebhookType: WebhookTypeDefault,
			Mode:        WebhookModeSpecific,
			Triggers: Triggers{
				{
					TriggerType: TriggerTypeStateChange,
					Condition:   map[string]interface{}{"state": model.CompletedState},
				},
			},
			WorkspaceID: ptrs.Ptr(int32(workspaceID)),
			Name:        webhookName2,
		}
		w3 := &Webhook{
			URL:         uuid.New().String(),
			WebhookType: WebhookTypeDefault,
			Mode:        WebhookModeSpecific,
			Triggers: Triggers{
				{
					TriggerType: TriggerTypeStateChange,
					Condition:   map[string]interface{}{"state": model.CompletedState},
				},
			},
			Name: webhookName3,
		}
		require.NoError(t, AddWebhook(ctx, w1))
		require.NoError(t, AddWebhook(ctx, w2))
		require.NoError(t, AddWebhook(ctx, w3))

		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			ID:        0,
			ProjectID: projectID,
			State:     model.CompletedState,
		}, config))

		require.Zero(t, countEventsForURL(ctx, t, w1.URL))
		require.Zero(t, countEventsForURL(ctx, t, w2.URL))
		require.Zero(t, countEventsForURL(ctx, t, w3.URL))

		config.RawIntegrations = &expconf.IntegrationsConfigV0{
			Webhooks: &expconf.WebhooksConfigV0{
				WebhookName: ptrs.Ptr([]string{webhookName2}),
			},
		}

		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			ID:        0,
			ProjectID: projectID,
			State:     model.CompletedState,
		}, config))

		require.Zero(t, countEventsForURL(ctx, t, w1.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w2.URL))
		require.Zero(t, countEventsForURL(ctx, t, w3.URL))

		config.RawIntegrations.Webhooks.WebhookName = ptrs.Ptr([]string{webhookName1, webhookName2, webhookName3})

		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			ID:        0,
			ProjectID: projectID,
			State:     model.CompletedState,
		}, config))

		require.Zero(t, countEventsForURL(ctx, t, w1.URL))
		require.Equal(t, 2, countEventsForURL(ctx, t, w2.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w3.URL))
	})

	clearWebhooksTables(ctx, t)
	t.Run("webhook with mode workspace and exclude", func(t *testing.T) {
		w1 := &Webhook{
			URL:         uuid.New().String(),
			WebhookType: WebhookTypeDefault,
			Mode:        WebhookModeWorkspace,
			Triggers: Triggers{
				{
					TriggerType: TriggerTypeStateChange,
					Condition:   map[string]interface{}{"state": model.CompletedState},
				},
			},
			WorkspaceID: ptrs.Ptr(int32(model.DefaultWorkspaceID)),
		}
		w2 := &Webhook{
			URL:         uuid.New().String(),
			WebhookType: WebhookTypeDefault,
			Mode:        WebhookModeWorkspace,
			Triggers: Triggers{
				{
					TriggerType: TriggerTypeStateChange,
					Condition:   map[string]interface{}{"state": model.CompletedState},
				},
			},
			WorkspaceID: ptrs.Ptr(int32(workspaceID)),
		}

		require.NoError(t, AddWebhook(ctx, w1))
		require.NoError(t, AddWebhook(ctx, w2))

		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			ID:        0,
			ProjectID: projectID,
			State:     model.CompletedState,
		}, config))

		require.Zero(t, countEventsForURL(ctx, t, w1.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w2.URL))

		config.RawIntegrations = &expconf.IntegrationsConfigV0{
			Webhooks: &expconf.WebhooksConfigV0{
				Exclude: true,
			},
		}

		require.NoError(t, ReportExperimentStateChanged(ctx, model.Experiment{
			ID:        0,
			ProjectID: projectID,
			State:     model.CompletedState,
		}, config))

		require.Zero(t, countEventsForURL(ctx, t, w1.URL))
		require.Equal(t, 1, countEventsForURL(ctx, t, w2.URL))
	})
}

var (
	testWebhookOne = Webhook{
		ID:          1000,
		URL:         "http://testwebhook.com",
		WebhookType: WebhookTypeSlack,
		Mode:        WebhookModeWorkspace,
	}
	testWebhookTwo = Webhook{
		ID:          2000,
		URL:         "http://testwebhooktwo.com",
		WebhookType: WebhookTypeDefault,
		Mode:        WebhookModeWorkspace,
	}
	testWebhookThree = Webhook{
		ID:          3000,
		URL:         "http://testwebhookthree.com",
		WebhookType: WebhookTypeSlack,
		Mode:        WebhookModeWorkspace,
	}
	testWebhookFour = Webhook{
		ID:          6000,
		URL:         "http://twebhook.com",
		WebhookType: WebhookTypeSlack,
		Mode:        WebhookModeWorkspace,
	}
	testWebhookFive = Webhook{
		ID:          7000,
		URL:         "http://twebhooktwo.com",
		WebhookType: WebhookTypeDefault,
		Mode:        WebhookModeWorkspace,
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
		Mode:        WebhookModeWorkspace,
	}
}

func TestDequeueEvents(t *testing.T) {
	ctx := context.Background()
	pgDB, closeDB := db.MustResolveTestPostgres(t)
	defer closeDB()
	db.MustMigrateTestPostgres(t, pgDB, db.MigrationsFromDB)
	clearWebhooksTables(ctx, t)

	singletonShipper = &shipper{wake: make(chan<- struct{})} // mock shipper

	workspaceName := uuid.New().String()
	workspaceID, _ := db.RequireMockWorkspaceID(t, pgDB, workspaceName)
	projectID, _ := db.RequireMockProjectID(t, pgDB, workspaceID, false)

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
		Mode:        WebhookModeWorkspace,
	}))

	t.Run("dequeueing and consuming a event should work", func(t *testing.T) {
		exp := model.Experiment{
			State:     model.CompletedState,
			ProjectID: projectID,
			ID:        0,
		}
		require.NoError(t, ReportExperimentStateChanged(ctx, exp, config))

		batch, err := dequeueEvents(ctx, maxEventBatchSize)
		require.NoError(t, batch.commit())
		require.NoError(t, err)
		require.Len(t, batch.events, 1)
	})

	t.Run("dequeueing and consuming a full batch of events should work", func(t *testing.T) {
		for i := 0; i < maxEventBatchSize; i++ {
			exp := model.Experiment{ID: 0, ProjectID: projectID, State: model.CompletedState}
			require.NoError(t, ReportExperimentStateChanged(ctx, exp, config))
		}

		batch, err := dequeueEvents(ctx, maxEventBatchSize)
		require.NoError(t, batch.commit())
		require.NoError(t, err)
		require.Len(t, batch.events, maxEventBatchSize)
	})

	t.Run("rolling back an event should work, and it should be reconsumed", func(t *testing.T) {
		exp := model.Experiment{ID: 0, ProjectID: projectID, State: model.CompletedState}
		require.NoError(t, ReportExperimentStateChanged(ctx, exp, config))

		batch, err := dequeueEvents(ctx, maxEventBatchSize)
		require.NoError(t, err)
		require.NoError(t, batch.rollback())

		batch, err = dequeueEvents(ctx, maxEventBatchSize)
		require.NoError(t, batch.commit())
		require.NoError(t, err)
		require.Len(t, batch.events, 1)
	})
}

func clearWebhooksTables(ctx context.Context, t *testing.T) {
	t.Log("clear webhooks db")
	_, err := db.Bun().NewDelete().Model((*Webhook)(nil)).Where("true").Exec(ctx)
	require.NoError(t, err)
	_, err = db.Bun().NewDelete().Model((*Event)(nil)).Where("true").Exec(ctx)
	require.NoError(t, err)
	_, err = db.Bun().NewDelete().Model((*webhookTaskLogTrigger)(nil)).Where("true").Exec(ctx)
	require.NoError(t, err)
}

func clearWebhooksEvent(ctx context.Context, t *testing.T) {
	t.Log("clear webhooks event")
	_, err := db.Bun().NewDelete().Model((*Event)(nil)).Where("true").Exec(ctx)
	require.NoError(t, err)
	_, err = db.Bun().NewDelete().Model((*webhookTaskLogTrigger)(nil)).Where("true").Exec(ctx)
	require.NoError(t, err)
}

func countEventsForURL(ctx context.Context, t *testing.T, url string) int {
	c, err := db.Bun().NewSelect().Model((*Event)(nil)).
		Where("url = ?", url).
		Count(ctx)
	require.NoError(t, err)

	return c
}
