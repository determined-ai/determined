//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

/*
A set of tests to ensure that the NTSC APIs call the expected AuthZ methods.
*/

var authzCommand *mocks.NSCAuthZ

func setupNTSCAuthzTest(t *testing.T) (
	*apiServer, *mocks.NSCAuthZ, model.User, context.Context,
) {
	api, curUser, ctx := setupAPITest(t)

	if authzCommand == nil {
		authzCommand = &mocks.NSCAuthZ{}
		command.AuthZProvider.Register("mock", authzCommand)
		config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}
	}
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}

	return api, authzCommand, curUser, ctx
}

func TestTasksCountAuthZ(t *testing.T) {
	api, authZCommand, curUser, ctx := setupNTSCAuthzTest(t)
	authZCommand.On("CanGetActiveTasksCount", mock.Anything, curUser).Return(fmt.Errorf("deny"))
	_, err := api.GetActiveTasksCount(ctx, &apiv1.GetActiveTasksCountRequest{})
	require.Equal(t, status.Error(codes.PermissionDenied, "deny"), err)
}

func TestAuthZCanGetNSC(t *testing.T) {
}

func TestAuthZCanTerminateNSC(t *testing.T) {
}

func TestAuthZCanCreateNSC(t *testing.T) {
}

func TestAuthZCanSetNSCsPriority(t *testing.T) {
}
