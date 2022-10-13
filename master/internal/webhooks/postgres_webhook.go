package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"

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

// GetWebhooks returns all Webhooks from the DB.
func GetWebhook(ctx context.Context, WebhookId int) (*Webhook, error) {
	webhook := Webhook{}
	err := db.Bun().NewSelect().
		Model(&webhook).
		Where("id = ?", WebhookId).
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

func CountEvents(ctx context.Context) (int, error) {
	return db.Bun().NewSelect().Model((*Event)(nil)).Count(ctx)
}

func ReportExperimentStateChanged(ctx context.Context, e model.Experiment) error {
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
		p, err := generateEventPayload(t.Webhook.WebhookType, e, e.State, TriggerTypeStateChange)
		if err != nil {
			return err
		}
		es = append(es, Event{Payload: p, TriggerID: t.ID, URL: t.Webhook.URL})
	}

	if _, err := db.Bun().NewInsert().Model(&es).Exec(ctx); err != nil {
		return err
	}

	singletonShipper.Wake()
	return nil
}

func generateEventPayload(wt WebhookType, e model.Experiment, expState model.State, tT TriggerType) ([]byte, error) {
	switch wt {
	case WebhookTypeDefault:
		expPayload := experimentToWebhookPayload(e)
		p := EventPayload{
			ID:        uuid.New(),
			Type:      tT,
			Timestamp: time.Now().Unix(),
			Condition: Condition{
				State: expState,
			},
			Data: EventData{
				Experiment: &expPayload,
			},
		}
		pJson, err := json.Marshal(p)
		if err != nil {
			return nil, err
		}
		return pJson, nil
	case WebhookTypeSlack:
		slackJson, err := generateSlackPayload(e)
		if err != nil {
			return nil, err
		}
		return slackJson, nil
	default:
		panic(fmt.Errorf("unknown webhook type: %+v", wt))
	}
}

func generateSlackPayload(e model.Experiment) ([]byte, error) {
	var status string
	var eUrl string
	var c string
	var mStatus string
	if e.State == model.CompletedState {
		status = "Your experiment completed successfully 🎉"

		eUrl = fmt.Sprintf("✅ %v (#%v)", e.Config.Name(), e.ID)
		c = "#13B670"
		mStatus = "Completed"
	} else {
		status = "Your experiment has stopped with errors"
		eUrl = fmt.Sprintf("❌ %v (#%v)", e.Config.Name(), e.ID)
		c = "#DD5040"
		mStatus = "Errored"
	}
	hours := e.EndTime.Sub(e.StartTime).Hours()
	hours, m := math.Modf(hours)
	minutes := int(m * 60)
	duration := fmt.Sprintf("%vh %vmin", hours, minutes)
	expBlockFields := []Field{
		{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Status*: %v", mStatus),
		},
		{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Duration*: %v", duration),
		},
	}
	if e.Config.Workspace() != "" {
		expBlockFields = append(expBlockFields, Field{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Workspace*: %v", e.Config.Workspace()),
		})
	}
	if e.Config.Project() != "" {
		expBlockFields = append(expBlockFields, Field{
			Type: "mrkdwn",
			Text: fmt.Sprintf("*Project*: %v", e.Config.Project()),
		})
	}
	experimentBlock := SlackBlock{
		Text: Field{
			Type: "mrkdwn",
			Text: eUrl,
		},
		Type:   "section",
		Fields: &expBlockFields,
	}
	messageBlock := SlackBlock{
		Text: Field{
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
		return nil, err
	}
	return message, nil
}

type eventBatch struct {
	tx       *bun.Tx
	events   []Event
	consumed bool
}

func (b *eventBatch) close() error {
	if !b.consumed {
		return b.tx.Rollback()
	}
	return nil
}

func (b *eventBatch) consume() error {
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
	DELETE FROM webhook_events_que
	USING ( SELECT * FROM webhook_events_que LIMIT ? FOR UPDATE SKIP LOCKED ) q
	WHERE q.id = webhook_events_que.id RETURNING webhook_events_que.*
`, limit).Scan(ctx, &events); err != nil {
		return nil, fmt.Errorf("scanning events: %w", err)
	}
	return &eventBatch{tx: &tx, events: events}, nil
}
