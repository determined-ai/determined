import { expect, test } from 'e2e/fixtures/global-fixtures';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';
import { safeName } from 'e2e/utils/naming';
import { V1Workspace } from 'services/api-ts-sdk';

test.describe('Workspace Model Registry', () => {
  let destinationWorkspace: V1Workspace;

  test.beforeAll(async ({ backgroundApiWorkspace }) => {
    destinationWorkspace = (
      await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new())
    ).workspace;
  });

  test.afterAll(async ({ backgroundApiWorkspace }) => {
    await backgroundApiWorkspace.deleteWorkspace(destinationWorkspace.id);
  });

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
      await expect(firstRow.name.pwLocator).toContainText(modelName);
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

    await test.step('Move', async () => {
      await (await firstRow.actions.open()).move.pwLocator.click();
      await modelRegistry.modelMoveModal.workspaceSelect.pwLocator.fill(destinationWorkspace.name);
      await modelRegistry.modelMoveModal.workspaceSelect.pwLocator.press('Enter');
      await modelRegistry.modelMoveModal.footer.submit.pwLocator.click();

      await modelRegistry.notification.pwLocator.waitFor();
      await expect(modelRegistry.notification.description.pwLocator).toContainText(
        `${modelName} moved to workspace ${destinationWorkspace.name}`,
      );

      await workspaceDetails.gotoWorkspace(destinationWorkspace.id);
      await workspaceDetails.modelRegistryTab.pwLocator.click();
      await expect(firstRow.name.pwLocator).toContainText(modelName);
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
