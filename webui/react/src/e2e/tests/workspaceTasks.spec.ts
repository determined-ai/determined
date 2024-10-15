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

      await firstRow.pwLocator.waitFor({ timeout: 10_000 });
    });

    await test.step('Kill task and copy task ID', async () => {
      await (await firstRow.actions.open()).kill.pwLocator.click();

      await workspaceDetails.taskList.taskKillModal.pwLocator.waitFor();
      await workspaceDetails.taskList.taskKillModal.killButton.pwLocator.click();
      await expect(firstRow.state.pwLocator).toHaveText('Terminated');

      await context.grantPermissions(['clipboard-read', 'clipboard-write']);
      await (await firstRow.actions.open()).copy.pwLocator.click();
      const handle = await authedPage.evaluateHandle(() => navigator.clipboard.readText());
      const clipboard = await handle.jsonValue();
      expect(clipboard.split('-').length).toBeGreaterThan(2);
      await expect(firstRow.taskID.pwLocator).toContainText(clipboard.split('-')[0]);
    });

    await test.step('View logs', async () => {
      await (await firstRow.actions.open()).viewLogs.pwLocator.click();

      const taskLogs = new TaskLogs(authedPage);

      await taskLogs.logViewer.pwLocator.waitFor();
      await taskLogs.logViewer.logEntry.nth(0).pwLocator.waitFor();
    });
  });
});
