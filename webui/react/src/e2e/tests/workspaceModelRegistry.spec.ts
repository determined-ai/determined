import { test } from 'e2e/fixtures/global-fixtures';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';

test.describe('Workspace Model Registry', () => {
  test('Model Registry', async ({ authedPage, newWorkspace }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);

    await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
    await workspaceDetails.modelRegistryTab.pwLocator.click();
  });
});
