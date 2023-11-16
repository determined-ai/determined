//go:build integration
// +build integration

package internal

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	apiPkg "github.com/determined-ai/determined/master/internal/api"
	authz2 "github.com/determined-ai/determined/master/internal/authz"
	"github.com/determined-ai/determined/master/internal/command"
	"github.com/determined-ai/determined/master/internal/config"
	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
)

/*
A set of tests to ensure that the NTSC APIs call the expected AuthZ methods.
*/

var authZNSC *mocks.NSCAuthZ

func setupNTSCAuthzTest(t *testing.T) (
	*apiServer, *mocks.NSCAuthZ, model.User, context.Context,
) {
	api, curUser, ctx := setupAPITest(t, nil)
	master := api.m

	cs, _ := command.NewService(master.db, master.rm)
	command.SetDefaultService(cs)

	jobservice.SetDefaultService(master.rm)

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
	genericNb, _ := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeNotebook,
		model.JobTypeNotebook,
		mockGenericReq(t, api.m.db))
	nb := genericNb.ToV1Notebook()

	_, err = api.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: invalidID})
	require.Equal(t, apiPkg.NotFoundErrs("notebook", invalidID, true), err)

	_, err = api.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: nb.Id})
	require.Equal(t, apiPkg.NotFoundErrs("notebook", nb.Id, true), err)

	// Commands.
	genericCmd, _ := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeCommand,
		model.JobTypeCommand,
		mockGenericReq(t, api.m.db))

	cmd := genericCmd.ToV1Command()

	_, err = api.GetCommand(ctx, &apiv1.GetCommandRequest{CommandId: invalidID})
	require.Equal(t, apiPkg.NotFoundErrs("command", invalidID, true), err)

	_, err = api.GetCommand(ctx, &apiv1.GetCommandRequest{CommandId: cmd.Id})
	require.Equal(t, apiPkg.NotFoundErrs("command", cmd.Id, true), err)

	// Shells.
	genericShell, _ := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeShell,
		model.JobTypeShell,
		mockGenericReq(t, api.m.db))
	shell := genericShell.ToV1Shell()

	_, err = api.GetShell(ctx, &apiv1.GetShellRequest{ShellId: invalidID})
	require.Equal(t, apiPkg.NotFoundErrs("shell", invalidID, true), err)

	_, err = api.GetShell(ctx, &apiv1.GetShellRequest{ShellId: shell.Id})
	require.Equal(t, apiPkg.NotFoundErrs("shell", shell.Id, true), err)

	// Tensorboards.
	// check permission errors are returned with not found status and follow the same pattern.
	authz.On("CanGetTensorboard", mock.Anything, curUser, mock.Anything, mock.Anything,
		mock.Anything).Return(authz2.PermissionDeniedError{}).Once()

	genericTb, _ := command.DefaultCmdService.LaunchGenericCommand(model.TaskTypeTensorboard,
		model.JobTypeTensorboard, mockGenericReq(t, api.m.db))
	tb := genericTb.ToV1Tensorboard()

	_, err = api.GetTensorboard(ctx, &apiv1.GetTensorboardRequest{TensorboardId: invalidID})
	require.Equal(t, apiPkg.NotFoundErrs("tensorboard", invalidID, true), err)

	_, err = api.GetTensorboard(ctx, &apiv1.GetTensorboardRequest{TensorboardId: tb.Id})
	require.Equal(t, apiPkg.NotFoundErrs("tensorboard", tb.Id, true), err)

	// check other errors are not returned with permission denied status.
	authz.On("CanGetNSC", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		errors.New("other error"),
	).Times(3)
	_, err = api.GetNotebook(ctx, &apiv1.GetNotebookRequest{NotebookId: nb.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
	require.NotEqual(t, codes.NotFound, status.Code(err))

	_, err = api.GetCommand(ctx, &apiv1.GetCommandRequest{CommandId: cmd.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
	require.NotEqual(t, codes.NotFound, status.Code(err))

	_, err = api.GetShell(ctx, &apiv1.GetShellRequest{ShellId: shell.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
	require.NotEqual(t, codes.NotFound, status.Code(err))

	authz.On("CanGetTensorboard", mock.Anything, curUser, mock.Anything, mock.Anything,
		mock.Anything).Return(errors.New("other error")).Once()

	_, err = api.GetTensorboard(ctx, &apiv1.GetTensorboardRequest{TensorboardId: tb.Id})
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
	genericNb, _ := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeNotebook,
		model.JobTypeNotebook,
		mockGenericReq(t, api.m.db))
	nb := genericNb.ToV1Notebook()

	_, err = api.KillNotebook(ctx, &apiv1.KillNotebookRequest{NotebookId: nb.Id})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Commands.
	genericCmd, _ := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeCommand,
		model.JobTypeCommand,
		mockGenericReq(t, api.m.db))
	cmd := genericCmd.ToV1Command()
	_, err = api.KillCommand(ctx, &apiv1.KillCommandRequest{CommandId: cmd.Id})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Shells.
	genericShell, _ := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeShell,
		model.JobTypeShell,
		mockGenericReq(t, api.m.db))
	shell := genericShell.ToV1Shell()
	_, err = api.KillShell(ctx, &apiv1.KillShellRequest{ShellId: shell.Id})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Tensorboards.
	authz.On("CanTerminateTensorboard", mock.Anything, curUser, mock.Anything).Return(
		authz2.PermissionDeniedError{},
	).Once()
	genericTb, _ := command.DefaultCmdService.LaunchGenericCommand(model.TaskTypeTensorboard,
		model.JobTypeTensorboard, mockGenericReq(t, api.m.db))
	tb := genericTb.ToV1Tensorboard()
	_, err = api.KillTensorboard(ctx, &apiv1.KillTensorboardRequest{TensorboardId: tb.Id})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// check other errors are not returned with permission denied status.
	authz.On("CanTerminateNSC", mock.Anything, curUser, mock.Anything).Return(
		errors.New("other error"),
	).Times(3)

	_, err = api.KillNotebook(ctx, &apiv1.KillNotebookRequest{NotebookId: nb.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.KillCommand(ctx, &apiv1.KillCommandRequest{CommandId: cmd.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.KillShell(ctx, &apiv1.KillShellRequest{ShellId: shell.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	authz.On("CanTerminateTensorboard", mock.Anything, curUser, mock.Anything).Return(
		errors.New("other error"),
	)
	_, err = api.KillTensorboard(ctx, &apiv1.KillTensorboardRequest{TensorboardId: tb.Id})
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
	genericNb, _ := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeNotebook,
		model.JobTypeNotebook,
		mockGenericReq(t, api.m.db))
	nb := genericNb.ToV1Notebook()
	_, err = api.SetNotebookPriority(ctx, &apiv1.SetNotebookPriorityRequest{NotebookId: nb.Id})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Commands.
	genericCmd, _ := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeCommand,
		model.JobTypeCommand,
		mockGenericReq(t, api.m.db))
	cmd := genericCmd.ToV1Command()
	_, err = api.SetCommandPriority(ctx, &apiv1.SetCommandPriorityRequest{CommandId: cmd.Id})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Shells.
	genericShell, _ := command.DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeShell,
		model.JobTypeShell,
		mockGenericReq(t, api.m.db))
	shell := genericShell.ToV1Shell()
	_, err = api.SetShellPriority(ctx, &apiv1.SetShellPriorityRequest{ShellId: shell.Id})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// Tensorboards.
	genericTb, _ := command.DefaultCmdService.LaunchGenericCommand(model.TaskTypeTensorboard,
		model.JobTypeTensorboard, mockGenericReq(t, api.m.db))
	tb := genericTb.ToV1Tensorboard()
	_, err = api.SetTensorboardPriority(ctx, &apiv1.SetTensorboardPriorityRequest{TensorboardId: tb.Id})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// check other errors are not returned with permission denied status.
	authz.On("CanSetNSCsPriority", mock.Anything, curUser, mock.Anything, mock.Anything).Return(
		errors.New("other error"),
	).Times(4)
	_, err = api.SetNotebookPriority(ctx, &apiv1.SetNotebookPriorityRequest{NotebookId: nb.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.SetCommandPriority(ctx, &apiv1.SetCommandPriorityRequest{CommandId: cmd.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.SetShellPriority(ctx, &apiv1.SetShellPriorityRequest{ShellId: shell.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))

	_, err = api.SetTensorboardPriority(ctx, &apiv1.SetTensorboardPriorityRequest{TensorboardId: tb.Id})
	require.NotNil(t, err)
	require.NotEqual(t, codes.PermissionDenied, status.Code(err))
}

func TestAuthZCanCreateNSC(t *testing.T) {
	api, authz, curUser, ctx := setupNTSCAuthzTest(t)
	var err error

	mockUserArg := mock.MatchedBy(func(u model.User) bool {
		return u.ID == curUser.ID
	})

	// check permission errors are returned with permission denied status.
	authz.On("CanCreateNSC", mock.Anything, mockUserArg, mock.Anything).Return(
		authz2.PermissionDeniedError{},
	).Times(3)
	_, err = api.LaunchNotebook(ctx, &apiv1.LaunchNotebookRequest{})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	_, err = api.LaunchCommand(ctx, &apiv1.LaunchCommandRequest{})
	require.Equal(t, codes.PermissionDenied, status.Code(err))
	_, err = api.LaunchShell(ctx, &apiv1.LaunchShellRequest{})
	require.Equal(t, codes.PermissionDenied, status.Code(err))

	// check other errors are not returned with permission denied status.
	authz.On("CanCreateNSC", mock.Anything, mockUserArg, mock.Anything).Return(
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

// HACK: duplicated from command package.
func mockGenericReq(t *testing.T, pgDB *db.PgDB) *command.CreateGeneric {
	user := db.RequireMockUser(t, pgDB)
	cmdSpec := tasks.GenericCommandSpec{}
	key := "pass"
	cmdSpec.Base = tasks.TaskSpec{
		Owner:  &model.User{ID: user.ID},
		TaskID: string(model.NewTaskID()),
	}
	cmdSpec.CommandID = uuid.New().String()
	cmdSpec.Metadata.PrivateKey = &key
	cmdSpec.Metadata.PublicKey = &key
	return &command.CreateGeneric{Spec: &cmdSpec}
}
