package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/uptrace/bun"

	conf "github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/workspace"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/schemas/expconf"
	"github.com/determined-ai/determined/proto/pkg/agentv1"

	"github.com/google/uuid"
)

// AddWebhook adds a Webhook and its Triggers to the DB.
func AddWebhook(ctx context.Context, w *Webhook) error {
	return db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(w).Exec(ctx)
		if err != nil {
			return err
		}
		for _, t := range w.Triggers {
			t.WebhookID = w.ID
		}

		if len(w.Triggers) != 0 {
			_, err = tx.NewInsert().Model(&w.Triggers).Exec(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetWebhook returns a single Webhooks from the DB.
func GetWebhook(ctx context.Context, webhookID int) (*Webhook, error) {
	webhook := Webhook{}
	err := db.Bun().NewSelect().
		Model(&webhook).
		Relation("Triggers").
		Where("id = ?", webhookID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return &webhook, nil
}

// GetWebhooks returns all Webhooks from the DB.
func GetWebhooks(ctx context.Context) (Webhooks, error) {
	webhooks := Webhooks{}
	err := db.Bun().NewSelect().
		Model(&webhooks).
		Relation("Triggers").
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return webhooks, nil
}

// DeleteWebhook deletes a Webhook and its Triggers from the DB.
func DeleteWebhook(ctx context.Context, id WebhookID) error {
	_, err := db.Bun().NewDelete().Model((*Webhook)(nil)).Where("id = ?", id).Exec(ctx)
	if err != nil {
		return err
	}
	return nil
}

// ReportHWFailure adds webhook event for an alert to the queue.
func ReportTaskAlert(ctx context.Context, alert agentv1.RunAlert) error {
	return nil
}

// ReportExperimentStateChanged adds webhook events to the queue.
// TODO(DET-8577): Remove unnecessary active config usage (remove the activeConfig parameter).
func ReportExperimentStateChanged(
	ctx context.Context, e model.Experiment, activeConfig expconf.ExperimentConfig,
) error {
	defer func() {
		if rec := recover(); rec != nil {
			log.Errorf("uncaught error in webhook report: %v", rec)
		}
	}()

	var ts []Trigger
	switch err := db.Bun().NewSelect().Model(&ts).Relation("Webhook").
		Where("trigger_type = ?", TriggerTypeStateChange).
		Where("condition->>'state' = ?", e.State).
		Scan(ctx); {
	case err != nil:
		return err
	case len(ts) == 0:
		return nil
	}

	var es []Event
	for _, t := range ts {
		p, err := generateEventPayload(
			ctx, t.Webhook.WebhookType, e, activeConfig, e.State, TriggerTypeStateChange,
		)
		if err != nil {
			return fmt.Errorf("error generating event payload: %w", err)
		}
		es = append(es, Event{Payload: p, URL: t.Webhook.URL})
	}
	if _, err := db.Bun().NewInsert().Model(&es).Exec(ctx); err != nil {
		return err
	}

	singletonShipper.Wake()
	return nil
}

func generateEventPayload(
	ctx context.Context,
	wt WebhookType,
	e model.Experiment,
	activeConfig expconf.ExperimentConfig,
	expState model.State,
	tT TriggerType,
) ([]byte, error) {
	switch wt {
	case WebhookTypeDefault:
		pJSON, err := json.Marshal(EventPayload{
			ID:        uuid.New(),
			Type:      tT,
			Timestamp: time.Now().Unix(),
			Condition: Condition{
				State: expState,
			},
			Data: EventData{
				Experiment: experimentToWebhookPayload(e, activeConfig),
			},
		})
		if err != nil {
			return nil, err
		}
		return pJSON, nil
	case WebhookTypeSlack:
		slackJSON, err := generateSlackPayload(ctx, e, activeConfig)
		if err != nil {
			return nil, err
		}
		return slackJSON, nil
	default:
		panic(fmt.Errorf("unknown webhook type: %+v", wt))
	}
}

func generateSlackPayload(
	ctx context.Context, e model.Experiment, activeConfig expconf.ExperimentConfig,
) ([]byte, error) {
	var status string
	var eURL string
	var c string
	var mStatus string
	var projectID int
	var wID int
	var w *model.Workspace
	config := conf.GetMasterConfig()
	wName := activeConfig.Workspace()
	pName := activeConfig.Project()
	webUIBaseURL := config.Webhooks.BaseURL
	baseURLIsSet := webUIBaseURL != ""
	if baseURLIsSet && wName != "" && pName != "" {
		ws, err := workspace.WorkspaceByName(ctx, wName)
		if err != nil {
			return nil, err
		}
		w = ws

		if w == nil {
			return nil, fmt.Errorf("unable to find workspace with name: %v", wName)
		}
		wID = w.ID

		pID, err := workspace.ProjectIDByName(ctx, wID, pName)
		if pID != nil {
			projectID = *pID
		}
		if err != nil {
			return nil, err
		}
	}

	if e.State == model.CompletedState {
		status = "Your experiment completed successfully 🎉"
		if baseURLIsSet {
			eURL = fmt.Sprintf("✅ <%v/det/experiments/%v/overview | %v (#%v)>",
				webUIBaseURL, e.ID, activeConfig.Name(), e.ID)
		} else {
			eURL = fmt.Sprintf("✅ %v (#%v)", activeConfig.Name(), e.ID)
		}
		c = "#13B670"
		mStatus = "Completed"
	} else {
		status = "Your experiment has stopped with errors"
		if baseURLIsSet {
			eURL = fmt.Sprintf("❌ <%v/det/experiments/%v/overview | %v (#%v)>",
				webUIBaseURL, e.ID, activeConfig.Name(), e.ID)
		} else {
			eURL = fmt.Sprintf("❌ %v (#%v)", activeConfig.Name(), e.ID)
		}
		c = "#DD5040"
		mStatus = "Errored"
	}
	hours := e.EndTime.Sub(e.StartTime).Hours()
	hours, m := math.Modf(hours)
	minutes := int(m * 60)
	duration := fmt.Sprintf("%vh %vmin", hours, minutes)
	expBlockFields := []SlackField{
		{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Status*: %v", mStatus),
		},
		{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Duration*: %v", duration),
		},
	}
	if wID != 0 && wName != "" && baseURLIsSet {
		expBlockFields = append(expBlockFields, SlackField{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Workspace*: <%v/det/workspaces/%v/projects | %v>",
				webUIBaseURL, wID, wName),
		})
	} else if wName != "" {
		expBlockFields = append(expBlockFields, SlackField{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Workspace*: %v", wName),
		})
	}
	if projectID != 0 && pName != "" && baseURLIsSet {
		expBlockFields = append(expBlockFields, SlackField{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Project*: <%v/det/projects/%v | %v>",
				webUIBaseURL, projectID, pName),
		})
	} else if pName != "" {
		expBlockFields = append(expBlockFields, SlackField{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Project*: %v", pName),
		})
	}
	experimentBlock := SlackBlock{
		Text: SlackField{
			Type: "mrkdwn",
			Text: eURL,
		},
		Type:   "section",
		Fields: &expBlockFields,
	}
	messageBlock := SlackBlock{
		Text: SlackField{
			Text: status,
			Type: "plain_text",
		},
		Type: "section",
	}
	attachment := SlackAttachment{
		Color:  c,
		Blocks: []SlackBlock{experimentBlock},
	}
	messageBody := SlackMessageBody{
		Blocks:      []SlackBlock{messageBlock},
		Attachments: &[]SlackAttachment{attachment},
	}

	message, err := json.Marshal(messageBody)
	if err != nil {
		return nil, fmt.Errorf("error creating slack payload: %w", err)
	}
	return message, nil
}

type eventBatch struct {
	tx       *bun.Tx
	events   []Event
	consumed bool
}

func (b *eventBatch) rollback() error {
	if !b.consumed {
		return b.tx.Rollback()
	}
	return nil
}

func (b *eventBatch) commit() error {
	b.consumed = true
	if err := b.tx.Commit(); err != nil {
		return fmt.Errorf("consuming event batch: %w", err)
	}
	return nil
}

func dequeueEvents(ctx context.Context, limit int) (*eventBatch, error) {
	tx, err := db.Bun().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	var events []Event
	if err = tx.NewRaw(`
DELETE FROM webhook_events_queue
USING ( SELECT * FROM webhook_events_queue LIMIT ? FOR UPDATE SKIP LOCKED ) q
WHERE q.id = webhook_events_queue.id RETURNING webhook_events_queue.*
`, limit).Scan(ctx, &events); err != nil {
		return nil, fmt.Errorf("scanning events: %w", err)
	}
	return &eventBatch{tx: &tx, events: events}, nil
}
