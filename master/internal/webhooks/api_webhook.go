package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/grpcutil"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// APIServer is an embedded api server struct.
type APIServer struct{}

// AuthorizeRequest checks if the user has CanEditWebhooks permissions.
func AuthorizeRequest(ctx context.Context) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}
	authErr := AuthZProvider.Get().
		CanEditWebhooks(curUser)
	if authErr != nil {
		return status.Error(codes.PermissionDenied, authErr.Error())
	}
	return nil
}

// GetWebhooks returns all Webhooks.
func (a *APIServer) GetWebhooks(
	ctx context.Context, req *apiv1.GetWebhooksRequest,
) (*apiv1.GetWebhooksResponse, error) {
	if err := AuthorizeRequest(ctx); err != nil {
		return nil, err
	}
	webhooks, err := GetWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetWebhooksResponse{Webhooks: webhooks.Proto()}, nil
}

// PostWebhook creates a new Webhook.
func (a *APIServer) PostWebhook(
	ctx context.Context, req *apiv1.PostWebhookRequest,
) (*apiv1.PostWebhookResponse, error) {
	if err := AuthorizeRequest(ctx); err != nil {
		return nil, err
	}
	if len(req.Webhook.Triggers) == 0 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"at least one trigger required",
		)
	}
	if _, err := url.ParseRequestURI(req.Webhook.Url); err != nil {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"valid url required",
		)
	}
	w := WebhookFromProto(req.Webhook)
	if err := AddWebhook(ctx, &w); err != nil {
		return nil, err
	}
	return &apiv1.PostWebhookResponse{Webhook: w.Proto()}, nil
}

// DeleteWebhook deletes a Webhook.
func (a *APIServer) DeleteWebhook(
	ctx context.Context, req *apiv1.DeleteWebhookRequest,
) (*apiv1.DeleteWebhookResponse, error) {
	if err := AuthorizeRequest(ctx); err != nil {
		return nil, err
	}
	if err := DeleteWebhook(ctx, WebhookID(req.Id)); err != nil {
		return nil, err
	}
	return &apiv1.DeleteWebhookResponse{}, nil
}

// TestWebhook sends a test event for a Webhook.
func (a *APIServer) TestWebhook(
	ctx context.Context, req *apiv1.TestWebhookRequest,
) (*apiv1.TestWebhookResponse, error) {
	if err := AuthorizeRequest(ctx); err != nil {
		return nil, err
	}
	webhook, err := GetWebhook(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	var tReq *http.Request
	switch webhook.WebhookType {
	case WebhookTypeDefault:
		s := "test"
		t := time.Now().Unix()
		tp := EventPayload{
			ID:        uuid.New(),
			Timestamp: t,
			Type:      TriggerTypeStateChange,
			Condition: Condition{
				State: "COMPLETED",
			},
			Data: EventData{
				TestData: &s,
			},
		}
		p, err := json.Marshal(tp)
		if err != nil {
			return nil, err
		}
		tReq, err = generateWebhookRequest(webhook.URL, p, t)
	case WebhookTypeSlack:
		testMessageBlock := SlackBlock{
			Text: Field{
				Text: "test",
				Type: "plain_text",
			},
			Type: "section",
		}
		messageBody := SlackMessageBody{
			Blocks: []SlackBlock{testMessageBlock},
		}
		slackMessage, err := json.Marshal(messageBody)
		if err != nil {
			return nil, err
		}
		tReq, err = http.NewRequest(
			http.MethodPost,
			webhook.URL,
			bytes.NewBuffer(slackMessage),
		)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "failed to create webhook request: %v ", err)
		}
	default:
		panic("Unknown webhook type")
	}
	if err != nil {
		return nil, err
	}
	c := http.Client{}
	resp, err := c.Do(tReq)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error sending webhook request: %s", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, status.Errorf(codes.InvalidArgument, "received error from webhook server: %v ", resp.StatusCode)
	}
	if err = resp.Body.Close(); err != nil {
		log.Error(fmt.Errorf("unable to close response body %v", err))
	}
	return &apiv1.TestWebhookResponse{}, nil
}
