package webhooks

import (
	"context"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/uptrace/bun"
	"gopkg.in/square/go-jose.v2/json"
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

func generateEventPayload(wt WebhookType, e model.Experiment) map[string]interface{} {
	var payload map[string]interface{}
	switch wt {
	case WebhookTypeDefault:
		payload = json.Marshal(e)
	case WebhookTypeSlack:
		panic("Not implemented")
	}
	return payload
}

func ReportExperimentStateChanged(ctx context.Context, e model.Experiment) error {
	// create webhook event model - DONE

	// get webhook types and trigger ids - DONE
	var triggers []Trigger

	err := db.Bun().NewSelect().Model(&triggers).Relation("Webhooks").Where("triggerType = ?", TriggerTypeStateChange).Where("condition ->> state = ?", e.State).Scan(ctx)
	if err != nil {
		return err
	}

	for _, trigger := range triggers {
		webhookType := trigger.Webhook.WebhookType

		// generate payload
		payload := generateEventPayload(webhookType, e)

		// generate model
		m := Event{
			Attempts:  0,
			Payload:   payload,
			TriggerID: trigger.ID,
		}

		// add to postgres
		_, err := db.Bun().NewInsert().Model(m).Exec(ctx)
	}

	// call something to wakeup the sender
}

func getEvents(ctx context.Context) ([]Event, error) {
	var events []Event
	err := db.Bun().NewSelect().Model(&events).Order("id ASC").Scan(ctx)
	if err != nil {
		return nil, err
	}
	return events, nil
}
