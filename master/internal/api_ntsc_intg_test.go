//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/pkg/errors"
	"testing"

	"github.com/determined-ai/determined/master/internal/task"

	"github.com/determined-ai/determined/proto/pkg/notebookv1"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

/*
A set of tests to ensure that the NTSC APIs call the expected AuthZ methods.
*/

var authZNSC *mocks.NSCAuthZ

func setupNTSCAuthzTest(t *testing.T) (
	*apiServer, *mocks.NSCAuthZ, model.User, context.Context,
) {
	api, curUser, ctx := setupAPITest(t)
	var master *Master
	master = api.m
	command.RegisterAPIHandler(
		master.system,
		nil,
		master.db,
		master.rm,
		&task.Logger{},
	)
	if authZNSC == nil {
		authZNSC = &mocks.NSCAuthZ{}
		command.AuthZProvider.Register("mock", authZNSC)
	}
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}

	return api, authZNSC, curUser, ctx
}

func TestTasksCountAuthZ(t *testing.T) {
	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
	authz.On("CanGetActiveTasksCount", mock.Anything, curUser).Return(fmt.Errorf("deny"))
	_, err := api.GetActiveTasksCount(ctx, &apiv1.GetActiveTasksCountRequest{})
	require.Equal(t, status.Error(codes.PermissionDenied, "deny"), err)
}

func TestCanGetNTSC(t *testing.T) {
	api, authz, curUser, ctx := setupNTSCAuthzTest(t)

	nbID := model.NewTaskID()
	var master *Master = api.m
	addr := notebooksAddr.Child(nbID)
	ref, newCreated := master.system.ActorOf(addr, actor.ActorFunc(func(context *actor.Context) error {
		nb := notebookv1.Notebook{Id: nbID.String()}
		if context.ExpectingResponse() {
			context.Respond(&apiv1.GetNotebookResponse{
				Notebook: &nb,
			})
		}
		return nil
	}))
	require.NotNil(t, ref)
	require.Equal(t, newCreated, true)

	// check permission errors are returned withe permission denied status.
	authz.On("CanGetNSC", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		false, nil,
	).Once()
	_, err := api.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: string(nbID)})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// check other errors are not returned withe permission denied status.
	authz.On("CanGetNSC", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		false, errors.New("other error"),
	)
	_, err = api.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: string(nbID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
}

func TestAuthZCanTerminateNSC(t *testing.T) {
	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
	nbID := model.NewTaskID()
	var master *Master = api.m
	addr := notebooksAddr.Child(nbID)
	ref, newCreated := master.system.ActorOf(addr, actor.ActorFunc(func(context *actor.Context) error {
		nb := notebookv1.Notebook{Id: nbID.String()}
		if context.ExpectingResponse() {
			context.Respond(&apiv1.GetNotebookResponse{
				Notebook: &nb,
			})
		}
		return nil
	}))
	require.NotNil(t, ref)
	require.Equal(t, newCreated, true)

	// check permission errors are returned withe permission denied status.
	authz.On("CanTerminateNSC", mock.Anything, curUser, mock.Anything).Return(
		&authz2.PermissionDeniedError{},
	).Once()
	_, err := api.KillNotebook(ctx, &apiv1.KillNotebookRequest{NotebookId: string(nbID)})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// check other errors are not returned withe permission denied status.
	authz.On("CanTerminateNSC", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		false, errors.New("other error"),
	)
	_, err = api.KillNotebook(ctx, &apiv1.KillNotebookRequest{NotebookId: string(nbID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
}

// func TestAuthZCanCreateNSC(t *testing.T) {
// 	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
// 	authz.On("CanCreateNSC", mock.Anything, curUser).Return(fmt.Errorf("deny"))
// 	_, err := api.GetActiveTasksCount(ctx, &apiv1.GetActiveTasksCountRequest{})
// 	require.Equal(t, status.Error(codes.PermissionDenied, "deny"), err)
// }

// func TestAuthZCanSetNSCsPriority(t *testing.T) {
// 	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
// 	authz.On("CanSetNSCsPriority", mock.Anything, curUser).Return(fmt.Errorf("deny"))
// 	_, err := api.GetActiveTasksCount(ctx, &apiv1.GetActiveTasksCountRequest{})
// 	require.Equal(t, status.Error(codes.PermissionDenied, "deny"), err)
// }
