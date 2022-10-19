package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/ptrs"

	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

// WebhooksAPIServer is an embedded api server struct.
type WebhooksAPIServer struct{}

// AuthorizeRequest checks if the user has CanEditWebhooks permissions.
// TODO remove this eventually since authz replaces this
// We can't yet since we use it else where.
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
func (a *WebhooksAPIServer) GetWebhooks(
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
func (a *WebhooksAPIServer) PostWebhook(
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
func (a *WebhooksAPIServer) DeleteWebhook(
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
func (a *WebhooksAPIServer) TestWebhook(
	ctx context.Context, req *apiv1.TestWebhookRequest,
) (*apiv1.TestWebhookResponse, error) {
	if err := AuthorizeRequest(ctx); err != nil {
		return nil, err
	}

	webhook, err := GetWebhook(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}

	eventID := uuid.New()
	log.Infof("creating webhook payload for event %v", eventID)

	var tReq *http.Request
	switch webhook.WebhookType {
	case WebhookTypeDefault:
		t := time.Now().Unix()
		p, perr := json.Marshal(EventPayload{
			ID:        uuid.New(),
			Timestamp: t,
			Type:      TriggerTypeStateChange,
			Condition: Condition{
				State: "COMPLETED",
			},
			Data: EventData{
				TestData: ptrs.Ptr("test"),
			},
		})
		if perr != nil {
			return nil, err
		}

		tr, rerr := generateWebhookRequest(ctx, webhook.URL, p, t)
		if rerr != nil {
			return nil, status.Errorf(codes.InvalidArgument,
				"failed to create webhook request for event %v error : %v ", eventID, err)
		}
		tReq = tr
	case WebhookTypeSlack:
		slackMessage, serr := json.Marshal(SlackMessageBody{
			Blocks: []SlackBlock{
				{
					Text: SlackField{
						Text: "test",
						Type: "plain_text",
					},
					Type: "section",
				},
			},
		})
		if serr != nil {
			return nil, err
		}

		tr, rerr := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			webhook.URL,
			bytes.NewBuffer(slackMessage),
		)
		if rerr != nil {
			return nil, status.Errorf(codes.InvalidArgument,
				"failed to create webhook request for event %v error : %v ", eventID, err)
		}
		tReq = tr
	default:
		panic("Unknown webhook type")
	}

	log.Infof("creating webhook request for event %v", eventID)
	c := http.Client{}
	resp, err := c.Do(tReq)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			"error sending webhook request for event %v error: %v", eventID, err)
	}
	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.WithError(err).Error("unable to close response body")
		}
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, status.Errorf(codes.InvalidArgument,
			"received error from webhook server for event %v error: %v ", eventID, resp.StatusCode)
	}
	return &apiv1.TestWebhookResponse{}, nil
}
