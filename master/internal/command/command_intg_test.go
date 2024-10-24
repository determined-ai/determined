//go:build integration
// +build integration

package command

import (
	"regexp"
	"testing"

	"github.com/determined-ai/determined/master/internal/user"

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

	// Launch a Command.
	cmd1 := launchCommand(t, db)

	// Get Command.
	resp1, err := DefaultCmdService.GetCommand(&apiv1.GetCommandRequest{CommandId: cmd1.Id})
	require.NotNil(t, resp1)
	require.NoError(t, err)

	// Launch another command.
	cmd2 := launchCommand(t, db)

	// Get Commands.
	resp2, err := DefaultCmdService.GetCommands(&apiv1.GetCommandsRequest{})
	require.NotNil(t, resp2)
	require.NoError(t, err)
	require.Len(t, resp2.Commands, 2)

	// Kill 1 command.
	resp3, err := DefaultCmdService.KillNTSC(cmd2.Id, model.TaskTypeCommand)
	require.NotNil(t, resp3)
	require.NoError(t, err)

	cmd3 := resp3.ToV1Command()
	require.Equal(t, taskv1.State_STATE_TERMINATED, cmd3.State)

	// Set command priority.
	resp4, err := DefaultCmdService.SetNTSCPriority(cmd1.Id, 0, model.TaskTypeCommand)
	require.NotNil(t, resp4)
	require.NoError(t, err)
}

func TestNotebookManagerLifecycle(t *testing.T) {
	db := setupTest(t)
	user.InitService(db, &model.ExternalSessions{})
	// Launch a Notebook.
	cmd1 := launchNotebook(t, db)

	// Get Notebook.
	resp1, err := DefaultCmdService.GetNotebook(&apiv1.GetNotebookRequest{NotebookId: cmd1.Id})
	require.NotNil(t, resp1)
	require.NoError(t, err)

	// Launch another Notebook.
	cmd2 := launchNotebook(t, db)
	// Get Notebooks.
	resp2, err := DefaultCmdService.GetNotebooks(&apiv1.GetNotebooksRequest{})
	require.NotNil(t, resp2)
	require.NoError(t, err)
	require.Len(t, resp2.Notebooks, 2)

	// Verify Notebooks return valid tokens.
	for _, resp := range resp2.Notebooks {
		re := regexp.MustCompile(`^/proxy/(.*)/\?token=(.*)`)
		addrMatches := re.FindStringSubmatch(resp.ServiceAddress)
		require.Len(t, addrMatches, 3)

		taskID, token := addrMatches[1], addrMatches[2]
		require.NotEmpty(t, token)
		require.NotEmpty(t, taskID)

		usr, notebookSession, err := user.GetService().UserAndNotebookSessionFromToken(token)
		require.NoError(t, err)
		require.Equal(t, resp.UserId, int32(usr.ID))
		require.Equal(t, taskID, string(notebookSession.TaskID))
	}

	// Kill 1 Notebook.
	resp3, err := DefaultCmdService.KillNTSC(cmd2.Id, model.TaskTypeNotebook)
	require.NotNil(t, resp3)
	require.NoError(t, err)

	nb3 := resp3.ToV1Notebook()
	require.Equal(t, taskv1.State_STATE_TERMINATED, nb3.State)

	// Set Notebook priority.
	resp4, err := DefaultCmdService.SetNTSCPriority(cmd1.Id, 0, model.TaskTypeNotebook)
	require.NotNil(t, resp4)
	require.NoError(t, err)
}

func TestShellManagerLifecycle(t *testing.T) {
	db := setupTest(t)

	// Launch a Shell.
	cmd1 := launchShell(t, db)

	// Get Shell.
	resp1, err := DefaultCmdService.GetShell(&apiv1.GetShellRequest{ShellId: cmd1.Id})
	require.NotNil(t, resp1)
	require.NoError(t, err)

	// Launch another Shell.
	cmd2 := launchShell(t, db)

	// Get Shells.
	resp2, err := DefaultCmdService.GetShells(&apiv1.GetShellsRequest{})
	require.NotNil(t, resp2)
	require.NoError(t, err)
	require.Len(t, resp2.Shells, 2)

	// Kill 1 Shell.
	resp3, err := DefaultCmdService.KillNTSC(cmd2.Id, model.TaskTypeShell)
	require.NotNil(t, resp3)
	require.NoError(t, err)

	shell3 := resp3.ToV1Shell()
	require.Equal(t, taskv1.State_STATE_TERMINATED, shell3.State)

	// Set Shell priority.
	resp4, err := DefaultCmdService.SetNTSCPriority(cmd1.Id, 0, model.TaskTypeShell)
	require.NotNil(t, resp4)
	require.NoError(t, err)
}

func TestTensorboardManagerLifecycle(t *testing.T) {
	db := setupTest(t)

	// Launch a Tensorboard.
	cmd1 := launchTensorboard(t, db)

	// Get Tensorboard.
	resp1, err := DefaultCmdService.GetTensorboard(&apiv1.GetTensorboardRequest{TensorboardId: cmd1.Id})
	require.NotNil(t, resp1)
	require.NoError(t, err)

	// Launch another Tensorboard.
	cmd2 := launchTensorboard(t, db)

	// Get Tensorboards.
	resp2, err := DefaultCmdService.GetTensorboards(&apiv1.GetTensorboardsRequest{})
	require.NotNil(t, resp2)
	require.NoError(t, err)
	require.Len(t, resp2.Tensorboards, 2)

	// Kill 1 Tensorboard.
	resp3, err := DefaultCmdService.KillNTSC(cmd2.Id, model.TaskTypeTensorboard)
	require.NotNil(t, resp3)
	require.NoError(t, err)

	tb3 := resp3.ToV1Tensorboard()
	require.Equal(t, taskv1.State_STATE_TERMINATED, tb3.State)

	// Set Tensorboard priority.
	resp4, err := DefaultCmdService.SetNTSCPriority(cmd1.Id, 0, model.TaskTypeTensorboard)
	require.NotNil(t, resp4)
	require.NoError(t, err)
}

func setupTest(t *testing.T) *db.PgDB {
	// First init the new Command Service
	var mockRM mocks.ResourceManager
	sub := sproto.NewAllocationSubscription(queue.New[sproto.ResourcesEvent](), func() {})
	mockRM.On("Allocate", mock.Anything, mock.Anything).Return(sub, nil)
	mockRM.On("Release", mock.Anything, mock.Anything).Return(nil)
	mockRM.On("SetGroupPriority", mock.Anything, mock.Anything).Return(nil)
	mockRM.On("SmallerValueIsHigherPriority").Return(true, nil)

	cs, _ := NewService(db.SingleDB(), &mockRM)
	SetDefaultService(cs)

	jobservice.SetDefaultService(&mockRM)

	require.NotNil(t, DefaultCmdService)
	return db.SingleDB()
}

func CreateMockGenericReq(t *testing.T, pgDB *db.PgDB) *CreateGeneric {
	user := db.RequireMockUser(t, pgDB)
	cmdSpec := tasks.GenericCommandSpec{}
	key := "pass"
	cmdSpec.Base = tasks.TaskSpec{
		Owner:        &model.User{ID: user.ID},
		TaskID:       string(model.NewTaskID()),
		ExtraEnvVars: map[string]string{},
	}
	cmdSpec.CommandID = uuid.New().String()
	cmdSpec.Metadata.PrivateKey = &key
	cmdSpec.Metadata.PublicKey = &key
	cmdSpec.Metadata.WorkspaceID = model.DefaultWorkspaceID
	cmdSpec.Base.Workspace = model.DefaultWorkspaceName
	return &CreateGeneric{Spec: &cmdSpec}
}

func launchCommand(t *testing.T, db *db.PgDB) *commandv1.Command {
	cmd, err := DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeCommand,
		model.JobTypeCommand,
		CreateMockGenericReq(t, db))
	v1cmd := cmd.ToV1Command()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.NotNil(t, DefaultCmdService.commands[model.TaskID(v1cmd.Id)])
	return v1cmd
}

func launchNotebook(t *testing.T, db *db.PgDB) *notebookv1.Notebook {
	mockReq := CreateMockGenericReq(t, db)
	cmd, err := DefaultCmdService.LaunchNotebookCommand(
		mockReq,
		mockReq.Spec.Base.Owner,
	)
	v1nb := cmd.ToV1Notebook()
	require.NoError(t, err)
	require.NotNil(t, v1nb)
	require.NotNil(t, DefaultCmdService.commands[model.TaskID(v1nb.Id)])
	return v1nb
}

func launchShell(t *testing.T, db *db.PgDB) *shellv1.Shell {
	cmd, err := DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeShell,
		model.JobTypeShell,
		CreateMockGenericReq(t, db))
	v1shell := cmd.ToV1Shell()
	require.NoError(t, err)
	require.NotNil(t, v1shell)
	require.NotNil(t, DefaultCmdService.commands[model.TaskID(v1shell.Id)])
	return v1shell
}

func launchTensorboard(t *testing.T, db *db.PgDB) *tensorboardv1.Tensorboard {
	cmd, err := DefaultCmdService.LaunchGenericCommand(
		model.TaskTypeTensorboard,
		model.JobTypeTensorboard,
		CreateMockGenericReq(t, db))
	v1tb := cmd.ToV1Tensorboard()
	require.NoError(t, err)
	require.NotNil(t, v1tb)
	require.NotNil(t, DefaultCmdService.commands[model.TaskID(v1tb.Id)])
	return v1tb
}
