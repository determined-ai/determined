import { expect, test } from 'e2e/fixtures/global-fixtures';
import { TaskLogs } from 'e2e/models/pages/TaskLogs';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';

test.describe('Workspace Tasks', () => {
  test('JupyterLab', async ({ authedPage, newWorkspace, context }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);
    const firstRow = workspaceDetails.taskList.table.table.rows.nth(0);

    await test.step('Start task', async () => {
      await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
      await workspaceDetails.tasksTab.pwLocator.click();

      await workspaceDetails.taskList.jupyterLabButton.pwLocator.click();
      await workspaceDetails.taskList.jupyterLabModal.pwLocator.waitFor();
      await workspaceDetails.taskList.jupyterLabModal.footer.submit.pwLocator.click();
      await workspaceDetails.taskList.jupyterLabModal.pwLocator.waitFor({ state: 'hidden' });

      const jupyterLabPage = await context.waitForEvent('page', { timeout: 10_000 });
      await jupyterLabPage.close();
    });

    await test.step('Kill task', async () => {
      await (await firstRow.actions.open()).kill.pwLocator.click();

      await workspaceDetails.taskList.taskKillModal.pwLocator.waitFor();
      await workspaceDetails.taskList.taskKillModal.killButton.pwLocator.click();

      await expect(firstRow.state.pwLocator).toHaveText('Terminated');
    });

    await test.step('View logs', async () => {
      await (await firstRow.actions.open()).viewLogs.pwLocator.click();

      const taskLogs = new TaskLogs(authedPage);

      await taskLogs.logViewer.pwLocator.waitFor();
      await taskLogs.logViewer.logEntry.nth(0).pwLocator.waitFor();
    });
  });
});
