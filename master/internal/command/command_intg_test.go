//go:build integration
// +build integration

package command

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/determined-ai/determined/master/internal/db"
	"github.com/determined-ai/determined/master/internal/job/jobservice"
	"github.com/determined-ai/determined/master/internal/mocks"
	"github.com/determined-ai/determined/master/internal/sproto"
	"github.com/determined-ai/determined/master/pkg/model"
	"github.com/determined-ai/determined/master/pkg/syncx/queue"
	"github.com/determined-ai/determined/master/pkg/tasks"
	"github.com/determined-ai/determined/proto/pkg/apiv1"
	"github.com/determined-ai/determined/proto/pkg/commandv1"
	"github.com/determined-ai/determined/proto/pkg/notebookv1"
	"github.com/determined-ai/determined/proto/pkg/shellv1"
	"github.com/determined-ai/determined/proto/pkg/taskv1"
	"github.com/determined-ai/determined/proto/pkg/tensorboardv1"
)

func TestCommandManagerLifecycle(t *testing.T) {
	db := setupTest(t)
	ctx := context.TODO()

	// Launch a Command.
	cmd1 := launchCommand(ctx, t, db)

	// Get Command.
	resp1, err := DefaultCmdService.GetCommand(&apiv1.GetCommandRequest{CommandId: cmd1.Id})
	require.NotNil(t, resp1)
	require.NoError(t, err)

	// Launch another command.
	cmd2 := launchCommand(ctx, t, db)

	// Get Commands.
	resp2, err := DefaultCmdService.GetCommands(&apiv1.GetCommandsRequest{})
	require.NotNil(t, resp2)
	require.NoError(t, err)
	require.Equal(t, len(resp2.Commands), 2)

	// Kill 1 command.
	resp3, err := DefaultCmdService.KillCommand(&apiv1.KillCommandRequest{CommandId: cmd2.Id})
	require.NotNil(t, resp3)
	require.NoError(t, err)
	require.Equal(t, resp3.Command.State, taskv1.State_STATE_TERMINATED)

	// Set command priority.
	resp4, err := DefaultCmdService.SetCommandPriority(&apiv1.SetCommandPriorityRequest{CommandId: cmd1.Id})
	require.NotNil(t, resp4)
	require.NoError(t, err)
}

func TestNotebookManagerLifecycle(t *testing.T) {
	db := setupTest(t)
	ctx := context.TODO()

	// Launch a Notebook.
	cmd1 := launchNotebook(ctx, t, db)

	// Get Notebook.
	resp1, err := DefaultCmdService.GetNotebook(&apiv1.GetNotebookRequest{NotebookId: cmd1.Id})
	require.NotNil(t, resp1)
	require.NoError(t, err)

	// Launch another Notebook.
	cmd2 := launchNotebook(ctx, t, db)

	// Get Notebooks.
	resp2, err := DefaultCmdService.GetNotebooks(&apiv1.GetNotebooksRequest{})
	require.NotNil(t, resp2)
	require.NoError(t, err)
	require.Equal(t, len(resp2.Notebooks), 2)

	// Kill 1 Notebook.
	resp3, err := DefaultCmdService.KillNotebook(&apiv1.KillNotebookRequest{NotebookId: cmd2.Id})
	require.NotNil(t, resp3)
	require.NoError(t, err)
	require.Equal(t, resp3.Notebook.State, taskv1.State_STATE_TERMINATED)

	// Set Notebook priority.
	resp4, err := DefaultCmdService.SetNotebookPriority(&apiv1.SetNotebookPriorityRequest{NotebookId: cmd1.Id})
	require.NotNil(t, resp4)
	require.NoError(t, err)
}

func TestShellManagerLifecycle(t *testing.T) {
	db := setupTest(t)
	ctx := context.TODO()

	// Launch a Shell.
	cmd1 := launchShell(ctx, t, db)

	// Get Shell.
	resp1, err := DefaultCmdService.GetShell(&apiv1.GetShellRequest{ShellId: cmd1.Id})
	require.NotNil(t, resp1)
	require.NoError(t, err)

	// Launch another Shell.
	cmd2 := launchShell(ctx, t, db)

	// Get Shells.
	resp2, err := DefaultCmdService.GetShells(&apiv1.GetShellsRequest{})
	require.NotNil(t, resp2)
	require.NoError(t, err)
	require.Equal(t, len(resp2.Shells), 2)

	// Kill 1 Shell.
	resp3, err := DefaultCmdService.KillShell(&apiv1.KillShellRequest{ShellId: cmd2.Id})
	require.NotNil(t, resp3)
	require.NoError(t, err)
	require.Equal(t, resp3.Shell.State, taskv1.State_STATE_TERMINATED)

	// Set Shell priority.
	resp4, err := DefaultCmdService.SetShellPriority(&apiv1.SetShellPriorityRequest{ShellId: cmd1.Id})
	require.NotNil(t, resp4)
	require.NoError(t, err)
}

func TestTensorboardManagerLifecycle(t *testing.T) {
	db := setupTest(t)
	ctx := context.TODO()

	// Launch a Tensorboard.
	cmd1 := launchTensorboard(ctx, t, db)

	// Get Tensorboard.
	resp1, err := DefaultCmdService.GetTensorboard(&apiv1.GetTensorboardRequest{TensorboardId: cmd1.Id})
	require.NotNil(t, resp1)
	require.NoError(t, err)

	// Launch another Tensorboard.
	cmd2 := launchTensorboard(ctx, t, db)

	// Get Tensorboards.
	resp2, err := DefaultCmdService.GetTensorboards(&apiv1.GetTensorboardsRequest{})
	require.NotNil(t, resp2)
	require.NoError(t, err)
	require.Equal(t, len(resp2.Tensorboards), 2)

	// Kill 1 Tensorboard.
	resp3, err := DefaultCmdService.KillTensorboard(&apiv1.KillTensorboardRequest{TensorboardId: cmd2.Id})
	require.NotNil(t, resp3)
	require.NoError(t, err)
	require.Equal(t, resp3.Tensorboard.State, taskv1.State_STATE_TERMINATED)

	// Set Tensorboard priority.
	resp4, err := DefaultCmdService.SetTensorboardPriority(&apiv1.SetTensorboardPriorityRequest{TensorboardId: cmd1.Id})
	require.NotNil(t, resp4)
	require.NoError(t, err)
}

func setupTest(t *testing.T) *db.PgDB {
	pgDB := db.MustSetupTestPostgres(t)
	// First init the new Command Service
	var mockRM mocks.ResourceManager
	sub := sproto.NewAllocationSubscription(queue.New[sproto.ResourcesEvent](), func() {})
	mockRM.On("Allocate", mock.Anything, mock.Anything).Return(sub, nil)
	mockRM.On("Release", mock.Anything, mock.Anything).Return(nil)
	mockRM.On("SetGroupPriority", mock.Anything, mock.Anything).Return(nil)

	SetDefaultCmdService(pgDB, &mockRM)
	jobservice.SetDefaultService(&mockRM)

	require.NotNil(t, DefaultCmdService)
	return pgDB
}

func CreateMockGenericReq(t *testing.T, pgDB *db.PgDB) *CreateGeneric {
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
	return &CreateGeneric{Spec: &cmdSpec}
}

func launchCommand(ctx context.Context, t *testing.T, db *db.PgDB) *commandv1.Command {
	cmd, err := DefaultCmdService.LaunchCommand(CreateMockGenericReq(t, db))
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotNil(t, DefaultCmdService.commands[model.TaskID(cmd.Id)])
	return cmd
}

func launchNotebook(ctx context.Context, t *testing.T, db *db.PgDB) *notebookv1.Notebook {
	notebook, err := DefaultCmdService.LaunchNotebook(CreateMockGenericReq(t, db))
	require.NoError(t, err)
	require.NotNil(t, notebook)
	require.NotNil(t, DefaultCmdService.commands[model.TaskID(notebook.Id)])
	return notebook
}

func launchShell(ctx context.Context, t *testing.T, db *db.PgDB) *shellv1.Shell {
	shell, err := DefaultCmdService.LaunchShell(CreateMockGenericReq(t, db))
	require.NoError(t, err)
	require.NotNil(t, shell)
	require.NotNil(t, DefaultCmdService.commands[model.TaskID(shell.Id)])
	return shell
}

func launchTensorboard(ctx context.Context, t *testing.T, db *db.PgDB) *tensorboardv1.Tensorboard {
	tensorboard, err := DefaultCmdService.LaunchTensorboard(CreateMockGenericReq(t, db))
	require.NoError(t, err)
	require.NotNil(t, tensorboard)
	require.NotNil(t, DefaultCmdService.commands[model.TaskID(tensorboard.Id)])
	return tensorboard
}
