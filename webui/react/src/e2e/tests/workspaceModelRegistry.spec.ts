import { expect, test } from 'e2e/fixtures/global-fixtures';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';
import { safeName } from 'e2e/utils/naming';

test.describe('Workspace Model Registry', () => {
  test('Model Registry', async ({ authedPage, newWorkspace }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);
    const modelRegistry = workspaceDetails.modelRegistry;
    const firstRow = modelRegistry.table.table.rows.nth(0);
    const modal = modelRegistry.modelCreateModal;
    const modelName = safeName('test-model');

    await test.step('Create model', async () => {
      await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
      await workspaceDetails.modelRegistryTab.pwLocator.click();

      await modelRegistry.newModelButton.pwLocator.click();
      await modal.pwLocator.waitFor();
      await modal.modelName.pwLocator.fill(modelName);
      await modal.footer.submit.pwLocator.click();
      await modal.pwLocator.waitFor({ state: 'hidden' });

      await modelRegistry.notification.pwLocator.waitFor();
      await expect(modelRegistry.notification.description.pwLocator).toContainText(
        `${modelName} has been created`,
      );
    });

    await test.step('Archive and Unarchive', async () => {
      await firstRow.pwLocator.waitFor();
      await (await firstRow.actions.open()).switchArchived.pwLocator.click();
      await modelRegistry.table.pwLocator.waitFor({ state: 'hidden' });

      await modelRegistry.showArchived.switch.pwLocator.click();
      await firstRow.archivedIcon.pwLocator.waitFor();

      await (await firstRow.actions.open()).switchArchived.pwLocator.click();
      await firstRow.archivedIcon.pwLocator.waitFor({ state: 'hidden' });
    });

    await test.step('Delete model', async () => {
      await firstRow.pwLocator.waitFor();
      await (await firstRow.actions.open()).delete.pwLocator.click();

      await modelRegistry.modelDeleteModal.pwLocator.waitFor();
      await modelRegistry.modelDeleteModal.deleteButton.pwLocator.click();

      await modelRegistry.table.pwLocator.waitFor({ state: 'hidden' });
    });
  });
});
