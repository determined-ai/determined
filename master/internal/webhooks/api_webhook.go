package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-cleanhttp"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/api"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/grpcutil"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/ptrs"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/webhookv1"
)

// WebhooksAPIServer is an embedded api server struct.
type WebhooksAPIServer struct{}

// authorizeEditRequest checks if the user has CanEditWebhooks permissions.
// TODO remove this eventually since authz replaces this
// We can't yet since we use it else where.
func authorizeEditRequest(ctx context.Context, workspaceID int32) error {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to get the user: %v", err)
	}
	var workspace *model.Workspace
	if workspaceID > 0 {
		workspace, err = getWorkspace(ctx, workspaceID)
		if err != nil {
			return err
		}
	}

	err = AuthZProvider.Get().CanEditWebhooks(ctx, curUser, workspace)
	if err != nil {
		return err
	}
	return nil
}

// GetWebhooks returns all Webhooks.
func (a *WebhooksAPIServer) GetWebhooks(
	ctx context.Context, req *apiv1.GetWebhooksRequest,
) (*apiv1.GetWebhooksResponse, error) {
	curUser, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get the user: %v", err)
	}
	workspaceIDs, err := AuthZProvider.Get().WebhookAvailableWorkspaces(ctx, curUser)
	if err != nil {
		return nil, err
	}
	webhooks, err := getWebhooks(ctx, &workspaceIDs)
	if err != nil {
		return nil, err
	}
	return &apiv1.GetWebhooksResponse{Webhooks: webhooks.Proto()}, nil
}

// PostWebhook creates a new Webhook.
func (a *WebhooksAPIServer) PostWebhook(
	ctx context.Context, req *apiv1.PostWebhookRequest,
) (*apiv1.PostWebhookResponse, error) {
	if err := authorizeEditRequest(ctx, req.Webhook.WorkspaceId); err != nil {
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

	for _, t := range req.Webhook.Triggers {
		if t.TriggerType == webhookv1.TriggerType_TRIGGER_TYPE_TASK_LOG {
			m := t.Condition.AsMap()
			if len(m) != 1 {
				return nil, status.Errorf(codes.InvalidArgument,
					"webhook task log condition must have one key got %v", m)
			}

			v, ok := m[regexConditionKey]
			if !ok {
				return nil, status.Errorf(codes.InvalidArgument,
					"webhook task log condition must have key '%s' got %v", regexConditionKey, m)
			}

			if _, typeOK := v.(string); !typeOK {
				return nil, status.Errorf(codes.InvalidArgument,
					"webhook task log condition must have key '%s' as string got %v",
					regexConditionKey, m)
			}
		}
		if t.TriggerType == webhookv1.TriggerType_TRIGGER_TYPE_CUSTOM {
			if req.Webhook.Mode != webhookv1.WebhookMode_WEBHOOK_MODE_SPECIFIC {
				return nil, status.Errorf(codes.InvalidArgument,
					"custom trigger only works on webhook with mode 'SPECIFIC'. Got %v",
					req.Webhook.Mode)
			}
		}
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
	webhook, err := GetWebhook(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	if err := authorizeEditRequest(ctx, webhook.Proto().WorkspaceId); err != nil {
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
	webhook, err := GetWebhook(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	if err := authorizeEditRequest(ctx, webhook.Proto().WorkspaceId); err != nil {
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
	c := cleanhttp.DefaultClient()
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

// PostWebhookEventData handles data for custom trigger.
func (a *WebhooksAPIServer) PostWebhookEventData(
	ctx context.Context, req *apiv1.PostWebhookEventDataRequest,
) (*apiv1.PostWebhookEventDataResponse, error) {
	var res apiv1.PostWebhookEventDataResponse
	_, _, err := grpcutil.GetUser(ctx)
	if err != nil {
		return &res, status.Errorf(codes.Internal, "failed to get the user: %s", err)
	}

	var data CustomTriggerData
	if req.Data != nil {
		data.Title = req.Data.Title
		data.Description = req.Data.Description
		data.Level = model.TaskLogLevelFromProto(req.Data.Level)
	}
	err = handleCustomTriggerData(ctx, data, int(req.ExperimentId), ptrs.Ptr(int(req.TrialId)))
	if err != nil {
		return &res, status.Errorf(codes.Internal,
			"failed to handle custom trigger data: %+v experiment id: %d trial_id %d : %s",
			data, req.ExperimentId, req.TrialId, err)
	}

	return &res, nil
}

func (a *WebhooksAPIServer) PatchWebhook(
	ctx context.Context, req *apiv1.PatchWebhookRequest,
) (*apiv1.PatchWebhookResponse, error) {
	webhook, err := GetWebhook(ctx, int(req.Id))
	if err != nil {
		return nil, err
	}
	if err := authorizeEditRequest(ctx, webhook.Proto().WorkspaceId); err != nil {
		return nil, err
	}

	err = UpdateWebhook(
		ctx,
		req.Id,
		req.Webhook,
	)
	if err != nil && errors.Is(err, db.ErrNotFound) {
		return nil, api.NotFoundErrs("webhook", strconv.Itoa(int(req.Id)), true)
	} else if err != nil {
		log.WithError(err).Errorf("failed to update webhook %d", req.Id)
		return nil, err
	}
	return &apiv1.PatchWebhookResponse{}, nil
}
