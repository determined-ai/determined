package internal

import (
	"github.com/determined-ai/determined/master/internal/rbac"
	"github.com/determined-ai/determined/master/internal/templates"
	"github.com/determined-ai/determined/master/internal/trials"
	"github.com/determined-ai/determined/master/internal/usergroup"
	"github.com/determined-ai/determined/master/internal/webhooks"
)

type apiServer struct {
	m *Master

	usergroup.UserGroupAPIServer
	rbac.RBACAPIServerWrapper
	webhooks.WebhooksAPIServer
	trials.TrialSourceInfoAPIServer
	trials.TrialsAPIServer
	templates.TemplateAPIServer
}
