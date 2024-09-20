import { test } from 'e2e/fixtures/global-fixtures';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';

test.describe('Workspace Tasks', () => {
  test('JupyterLab', async ({ authedPage, newWorkspace, context }) => {
    const pagePromise = context.waitForEvent('page');
    const workspaceDetails = new WorkspaceDetails(authedPage);

    await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
    await workspaceDetails.tasksTab.pwLocator.click();

    await workspaceDetails.taskList.jupyterLabButton.pwLocator.click();
    await workspaceDetails.taskList.jupyterLabModal.pwLocator.waitFor();
    await workspaceDetails.taskList.jupyterLabModal.footer.submit.pwLocator.click();
    await workspaceDetails.taskList.jupyterLabModal.pwLocator.waitFor({ state: 'hidden' });

    const jupyterLabPage = await pagePromise;
    await jupyterLabPage.close();

    await workspaceDetails.taskList.table.pwLocator.waitFor();
    const firstRow = await workspaceDetails.taskList.table.table.rows.nth(0);
    await (await firstRow.actions.open()).kill.pwLocator.click();

    await workspaceDetails.taskList.taskKillModal.pwLocator.waitFor();
    await workspaceDetails.taskList.taskKillModal.killButton.pwLocator.click();

    await (await firstRow.actions.open()).viewLogs.pwLocator.click();

    // confirm state
    // view logs
  });
});
