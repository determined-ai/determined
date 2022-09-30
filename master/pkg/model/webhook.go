package model

import (
	"github.com/uptrace/bun"
	"github.com/determined-ai/determined/proto/pkg/webhookv1"
	"google.golang.org/protobuf/types/known/structpb"
)


// WebhookID is the type for Webhook IDs.
type WebhookID int

// TriggerID is the type for user Trigger IDs.
type TriggerID int

// Webhook corresponds to a row in the "webhooks" DB table.
type Webhook struct {
	bun.BaseModel           `bun:"table:webhooks"`
	ID            WebhookID    `db:"id" json:"id"`
	Url           string           `db:"url" json:"url"`
}

// Trigger corresponds to a row in the "webhook_triggers" DB table.
type Trigger struct {
	bun.BaseModel           `bun:"table:webhook_triggers"`
	ID            TriggerID `db:"id" json:"id"`
	TriggerType   webhookv1.TriggerType    `db:"trigger_type" json:"trigger_type"`
	Condition     *structpb.Struct `db:"condition" json:"condition"`
	WebhookId    WebhookID `db:"webhook_id" json:"webhook_id"`
}

type WebhookWithTriggers struct {
	ID            WebhookID `json:"id"`
	Url     	  string     `json:"url"`
	Triggers      []Trigger `json:"triggers"`
}

// Proto converts a user to its protobuf representation.
func (webhook *Webhook) Proto() *webhookv1.Webhook {
	return &webhookv1.Webhook{
		Id:          int32(webhook.ID),
		Url:    	 webhook.Url,
	}
}

// Proto converts a user to its protobuf representation.
func (trigger *Trigger) Proto() *webhookv1.Trigger {
	return &webhookv1.Trigger{
		Id:          int32(trigger.ID),
		TriggerType:  trigger.TriggerType,
		Condition:  trigger.Condition,
		WebhookId:  int32(trigger.WebhookId),
	}
}

func (webhookWithTriggers *WebhookWithTriggers) Proto() *webhookv1.WebhookWithTriggers {
	triggers := []*webhookv1.Trigger{}
	for _, trigger := range webhookWithTriggers.Triggers {
		protoTrigger := webhookv1.Trigger{}
		protoTrigger.Id = int32(trigger.ID)
		protoTrigger.Condition = trigger.Condition
		protoTrigger.WebhookId = int32(trigger.WebhookId)
		protoTrigger.TriggerType = trigger.TriggerType
		triggers = append(triggers, &protoTrigger)
	}
	return &webhookv1.WebhookWithTriggers{
		Id:          int32(webhookWithTriggers.ID),
		Url:         webhookWithTriggers.Url,
		Triggers:  	 triggers,
	}
}

// Webhooks is a slice of Webhook objects.
type Webhooks []Webhook

// Triggers is a slice of Trigger objects—primarily useful for its methods.
type Triggers []Trigger

// Proto converts a slice of webhooks to its protobuf representation.
func (webhooks Webhooks) Proto() []*webhookv1.Webhook {
	out := make([]*webhookv1.Webhook, len(webhooks))
	for i, w := range webhooks {
		out[i] = w.Proto()
	}
	return out
}

// Proto converts a slice of triggers to its protobuf representation.
func (triggers Triggers) Proto() []*webhookv1.Trigger {
	out := make([]*webhookv1.Trigger, len(triggers))
	for i, t := range triggers {
		out[i] = t.Proto()
	}
	return out
}
