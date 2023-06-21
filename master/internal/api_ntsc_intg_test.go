//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/actor"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
	"github.com/determined-ai/determined/proto/pkg/shellv1"
)

/*
A set of tests to ensure that the NTSC APIs call the expected AuthZ methods.
*/

var authZNSC *mocks.NSCAuthZ

func setupNTSCAuthzTest(t *testing.T) (
	*apiServer, *mocks.NSCAuthZ, model.User, context.Context,
) {
	api, curUser, ctx := setupAPITest(t, nil)
	var master *Master = api.m
	command.RegisterAPIHandler(
		master.system,
		nil,
		master.db,
		master.rm,
	)
	authZNSC = &mocks.NSCAuthZ{}
	command.AuthZProvider.RegisterOverride("mock", authZNSC)
	config.GetMasterConfig().Security.AuthZ = config.AuthZConfig{Type: "mock"}
	return api, authZNSC, curUser, ctx
}

func setupNSCAuthZ() *mocks.NSCAuthZ {
	if authZNSC == nil {
		authZNSC = &mocks.NSCAuthZ{}
		command.AuthZProvider.RegisterOverride("mock", authZNSC)
	}
	return authZNSC
}

func setupMockNBActor(t *testing.T, master *Master) model.TaskID {
	nbID := model.NewTaskID()
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
	return nbID
}

func setupMockCMDActor(t *testing.T, master *Master) model.TaskID {
	cmdID := model.NewTaskID()
	addr := commandsAddr.Child(cmdID)
	ref, newCreated := master.system.ActorOf(addr, actor.ActorFunc(func(context *actor.Context) error {
		cmd := commandv1.Command{Id: cmdID.String()}
		if context.ExpectingResponse() {
			context.Respond(&apiv1.GetCommandResponse{
				Command: &cmd,
			})
		}
		return nil
	}))
	require.NotNil(t, ref)
	require.Equal(t, newCreated, true)
	return cmdID
}

func setupMockShellActor(t *testing.T, master *Master) model.TaskID {
	shellID := model.NewTaskID()
	addr := shellsAddr.Child(shellID)
	ref, newCreated := master.system.ActorOf(addr, actor.ActorFunc(func(context *actor.Context) error {
		shell := shellv1.Shell{Id: shellID.String()}
		if context.ExpectingResponse() {
			context.Respond(&apiv1.GetShellResponse{
				Shell: &shell,
			})
		}
		return nil
	}))
	require.NotNil(t, ref)
	require.Equal(t, true, newCreated)
	return shellID
}

func setupMockTensorboardActor(t *testing.T, master *Master) model.TaskID {
	tbID := model.NewTaskID()
	addr := tensorboardsAddr.Child(tbID)
	ref, newCreated := master.system.ActorOf(addr, actor.ActorFunc(func(context *actor.Context) error {
		tb := tensorboardv1.Tensorboard{Id: tbID.String()}
		if context.ExpectingResponse() {
			context.Respond(&apiv1.GetTensorboardResponse{
				Tensorboard: &tb,
			})
		}
		return nil
	}))
	require.NotNil(t, ref)
	require.Equal(t, true, newCreated)
	return tbID
}

func TestTasksCountAuthZ(t *testing.T) {
	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
	authz.On("CanGetActiveTasksCount", mock.Anything, curUser).Return(fmt.Errorf("deny"))
	_, err := api.GetActiveTasksCount(ctx, &apiv1.GetActiveTasksCountRequest{})
	require.Equal(t, status.Error(codes.PermissionDenied, "deny"), err)
}

func TestCanGetNTSC(t *testing.T) {
	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
	var err error

	// check permission errors are returned with not found status and follow the same pattern.
	authz.On("CanGetNSC", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		authz2.PermissionDeniedError{}).Times(3)

	invalidID := "non-existing"

	// Notebooks.
	nbID := setupMockNBActor(t, api.m)
	nbsActor := actor.Addr(command.NotebookActorPath)

	_, err = api.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: invalidID})
	require.Equal(t, apiPkg.NotFoundErrs("actor", fmt.Sprint(nbsActor.Child(invalidID)), true), err)

	_, err = api.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: string(nbID)})
	require.Equal(t, apiPkg.NotFoundErrs("actor", fmt.Sprint(nbsActor.Child(nbID)), true), err)

	// Commands.
	cmdID := setupMockCMDActor(t, api.m)
	cmdsActor := actor.Addr(command.CommandActorPath)

	_, err = api.GetCommand(ctx, &apiv1.GetCommandRequest{CommandId: invalidID})
	require.Equal(t, apiPkg.NotFoundErrs("actor", fmt.Sprint(cmdsActor.Child(invalidID)), true), err)

	_, err = api.GetCommand(ctx, &apiv1.GetCommandRequest{CommandId: string(cmdID)})
	require.Equal(t, apiPkg.NotFoundErrs("actor", fmt.Sprint(cmdsActor.Child(cmdID)), true), err)

	// Shells.
	shellID := setupMockShellActor(t, api.m)
	shellsActor := actor.Addr(command.ShellActorPath)

	_, err = api.GetShell(ctx, &apiv1.GetShellRequest{ShellId: invalidID})
	require.Equal(t, apiPkg.NotFoundErrs("actor",
		fmt.Sprint(shellsActor.Child(invalidID)), true), err)

	_, err = api.GetShell(ctx, &apiv1.GetShellRequest{ShellId: string(shellID)})
	require.Equal(t, apiPkg.NotFoundErrs("actor", fmt.Sprint(shellsActor.Child(shellID)), true), err)

	// Tensorboards.
	// check permission errors are returned with not found status and follow the same pattern.
	authz.On("CanGetTensorboard", mock.Anything, curUser, mock.Anything, mock.Anything,
		mock.Anything).Return(authz2.PermissionDeniedError{}).Once()

	tbID := setupMockTensorboardActor(t, api.m)
	tbActor := actor.Addr(command.TensorboardActorPath)

	_, err = api.GetTensorboard(ctx, &apiv1.GetTensorboardRequest{TensorboardId: invalidID})
	require.Equal(t, apiPkg.NotFoundErrs("actor", fmt.Sprint(tbActor.Child(invalidID)), true), err)

	_, err = api.GetTensorboard(ctx, &apiv1.GetTensorboardRequest{TensorboardId: string(tbID)})
	require.Equal(t, apiPkg.NotFoundErrs("actor", fmt.Sprint(tbActor.Child(tbID)), true), err)

	// check other errors are not returned with permission denied status.
	authz.On("CanGetNSC", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		errors.New("other error"),
	).Times(3)
	_, err = api.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: string(nbID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
	require.NotEqual(t, codes.NotFound, status.Code(err))

	_, err = api.GetCommand(ctx, &apiv1.GetCommandRequest{CommandId: string(cmdID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
	require.NotEqual(t, codes.NotFound, status.Code(err))

	_, err = api.GetShell(ctx, &apiv1.GetShellRequest{ShellId: string(shellID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
	require.NotEqual(t, codes.NotFound, status.Code(err))

	authz.On("CanGetTensorboard", mock.Anything, curUser, mock.Anything, mock.Anything,
		mock.Anything).Return(errors.New("other error")).Once()

	_, err = api.GetTensorboard(ctx, &apiv1.GetTensorboardRequest{TensorboardId: string(tbID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
	require.NotEqual(t, codes.NotFound, status.Code(err))
}

func TestAuthZCanTerminateNSC(t *testing.T) {
	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
	var err error
	authz.On("CanGetNSC", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		nil,
	)
	authz.On("CanGetTensorboard", mock.Anything, curUser, mock.Anything, mock.Anything,
		mock.Anything).Return(nil)

	// check permission errors are returned with permission denied status.
	authz.On("CanTerminateNSC", mock.Anything, curUser, mock.Anything).Return(
		authz2.PermissionDeniedError{},
	).Times(3)

	// Notebooks.
	nbID := setupMockNBActor(t, api.m)
	_, err = api.KillNotebook(ctx, &apiv1.KillNotebookRequest{NotebookId: string(nbID)})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Commands.
	cmdID := setupMockCMDActor(t, api.m)
	_, err = api.KillCommand(ctx, &apiv1.KillCommandRequest{CommandId: string(cmdID)})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Shells.
	shellID := setupMockShellActor(t, api.m)
	_, err = api.KillShell(ctx, &apiv1.KillShellRequest{ShellId: string(shellID)})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Tensorboards.
	authz.On("CanTerminateTensorboard", mock.Anything, curUser, mock.Anything).Return(
		authz2.PermissionDeniedError{},
	).Once()
	tbID := setupMockTensorboardActor(t, api.m)
	_, err = api.KillTensorboard(ctx, &apiv1.KillTensorboardRequest{TensorboardId: string(tbID)})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// check other errors are not returned with permission denied status.
	authz.On("CanTerminateNSC", mock.Anything, curUser, mock.Anything).Return(
		errors.New("other error"),
	).Times(3)

	_, err = api.KillNotebook(ctx, &apiv1.KillNotebookRequest{NotebookId: string(nbID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.KillCommand(ctx, &apiv1.KillCommandRequest{CommandId: string(cmdID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.KillShell(ctx, &apiv1.KillShellRequest{ShellId: string(shellID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	authz.On("CanTerminateTensorboard", mock.Anything, curUser, mock.Anything).Return(
		errors.New("other error"),
	)
	_, err = api.KillTensorboard(ctx, &apiv1.KillTensorboardRequest{TensorboardId: string(tbID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
}

func TestAuthZCanSetNSCsPriority(t *testing.T) {
	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
	var err error
	authz.On("CanGetNSC", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		nil,
	)
	authz.On("CanGetTensorboard", mock.Anything, curUser, mock.Anything, mock.Anything,
		mock.Anything).Return(nil)

	// check permission errors are returned with permission denied status.
	authz.On("CanSetNSCsPriority", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		authz2.PermissionDeniedError{},
	).Times(4)

	// Notebooks.
	nbID := setupMockNBActor(t, api.m)
	_, err = api.SetNotebookPriority(ctx, &apiv1.SetNotebookPriorityRequest{NotebookId: string(nbID)})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Commands.
	cmdID := setupMockCMDActor(t, api.m)
	_, err = api.SetCommandPriority(ctx, &apiv1.SetCommandPriorityRequest{CommandId: string(cmdID)})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Shells.
	shellID := setupMockShellActor(t, api.m)
	_, err = api.SetShellPriority(ctx, &apiv1.SetShellPriorityRequest{ShellId: string(shellID)})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Tensorboards.
	tbID := setupMockTensorboardActor(t, api.m)
	_, err = api.SetTensorboardPriority(ctx, &apiv1.SetTensorboardPriorityRequest{
		TensorboardId: string(tbID),
	})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// check other errors are not returned with permission denied status.
	authz.On("CanSetNSCsPriority", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		errors.New("other error"),
	).Times(4)
	_, err = api.SetNotebookPriority(ctx, &apiv1.SetNotebookPriorityRequest{NotebookId: string(nbID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.SetCommandPriority(ctx, &apiv1.SetCommandPriorityRequest{CommandId: string(cmdID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.SetShellPriority(ctx, &apiv1.SetShellPriorityRequest{ShellId: string(shellID)})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.SetTensorboardPriority(ctx, &apiv1.SetTensorboardPriorityRequest{
		TensorboardId: string(tbID),
	})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
}

func TestAuthZCanCreateNSC(t *testing.T) {
	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
	var err error

	// check permission errors are returned with permission denied status.
	authz.On("CanCreateNSC", mock.Anything, curUser, mock.Anything).Return(
		authz2.PermissionDeniedError{},
	).Times(3)
	_, err = api.LaunchNotebook(ctx, &apiv1.LaunchNotebookRequest{})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	_, err = api.LaunchCommand(ctx, &apiv1.LaunchCommandRequest{})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	_, err = api.LaunchShell(ctx, &apiv1.LaunchShellRequest{})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// check other errors are not returned with permission denied status.
	authz.On("CanCreateNSC", mock.Anything, curUser, mock.Anything).Return(
		errors.New("other error"),
	).Times(3)
	_, err = api.LaunchNotebook(ctx, &apiv1.LaunchNotebookRequest{})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
	_, err = api.LaunchCommand(ctx, &apiv1.LaunchCommandRequest{})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
	_, err = api.LaunchShell(ctx, &apiv1.LaunchShellRequest{})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
}
