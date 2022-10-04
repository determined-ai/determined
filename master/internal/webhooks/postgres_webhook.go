package webhooks

import (
	"context"

	"github.com/uptrace/bun"

	"github.com/determined-ai/determined/master/internal/db"
)

func AddWebhook(ctx context.Context, w *Webhook) error {
	return db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewInsert().Model(w).Exec(ctx)
		if err != nil {
			return err
		}
		for _, t := range w.Triggers {
			t.WebhookId = w.ID
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

func DeleteWebhook(ctx context.Context, id WebhookID) error {
	return db.Bun().RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := tx.NewDelete().Model((*Webhook)(nil)).Where("id = ?", id).Exec(ctx)
		if err != nil {
			return err
		}

		_, err = tx.NewDelete().Model((*Trigger)(nil)).Where("webhook_id = ?", id).Exec(ctx)
		if err != nil {
			return err
		}
		return nil
	})
	return nil
}
