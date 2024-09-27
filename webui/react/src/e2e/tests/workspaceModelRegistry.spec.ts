import { expect, test } from 'e2e/fixtures/global-fixtures';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';
import { safeName } from 'e2e/utils/naming';
import { V1Workspace } from 'services/api-ts-sdk';

test.describe('Workspace Model Registry', () => {
  const workspaces = new Map<'origin' | 'destination', V1Workspace>();
  const modelName = safeName('test-model');

  test.beforeAll(async ({ backgroundAuthedPage, newWorkspace }) => {
    const workspaceDetails = new WorkspaceDetails(backgroundAuthedPage);
    const modelRegistry = workspaceDetails.modelRegistry;
    const firstRow = modelRegistry.table.table.rows.nth(0);
    const modal = modelRegistry.modelCreateModal;

    workspaces.set('origin', newWorkspace.response.workspace);

    await workspaceDetails.gotoWorkspace(workspaces.get('origin')?.id);
    await workspaceDetails.modelRegistryTab.pwLocator.click();

    await modelRegistry.newModelButton.pwLocator.click();

    await modal.name.pwLocator.fill(modelName);
    await modal.description.pwLocator.fill(modelName + ' description');

    await modal.addMoreDetails.pwLocator.click();
    await modal.addMetadatButton.pwLocator.click();
    await modal.addTagButton.pwLocator.click();

    await modal.metadataKey.pwLocator.fill('metadata_key');
    await modal.metadataValue.pwLocator.fill('metadata_value');
    await modal.tag.pwLocator.fill('tag');

    await modal.footer.submit.pwLocator.click();

    await expect(modelRegistry.notification.description.pwLocator).toContainText(
      `${modelName} has been created`,
    );

    await backgroundAuthedPage.reload();
    await expect(firstRow.name.pwLocator).toContainText(modelName);
  });

  test.afterAll(async ({ backgroundApiWorkspace, backgroundAuthedPage }) => {
    const workspaceDetails = new WorkspaceDetails(backgroundAuthedPage);
    const modelRegistry = workspaceDetails.modelRegistry;
    const firstRow = modelRegistry.table.table.rows.nth(0);

    await test.step('Delete model', async () => {
      const workspace = workspaces.get('destination') ?? workspaces.get('origin');

      await workspaceDetails.gotoWorkspace(workspace?.id);
      await workspaceDetails.modelRegistryTab.pwLocator.click();

      await (await firstRow.actions.open()).delete.pwLocator.click();
      await modelRegistry.modelDeleteModal.deleteButton.pwLocator.click();

      await workspaceDetails.gotoWorkspace(workspace?.id);
      await workspaceDetails.modelRegistryTab.pwLocator.click();

      await modelRegistry.noModelsMessage.pwLocator.waitFor();
    });

    await test.step('Delete destination workspace', async () => {
      const destinationWorkspace = workspaces.get('destination');
      if (destinationWorkspace) {
        await backgroundApiWorkspace.deleteWorkspace(destinationWorkspace.id);
      }
    });
  });

  test('Archive and Unarchive', async ({ authedPage, newWorkspace }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);
    const modelRegistry = workspaceDetails.modelRegistry;
    const firstRow = modelRegistry.table.table.rows.nth(0);

    await test.step('Archive', async () => {
      await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
      await workspaceDetails.modelRegistryTab.pwLocator.click();

      await (await firstRow.actions.open()).switchArchived.pwLocator.click();
      await modelRegistry.noModelsMessage.pwLocator.waitFor();

      await modelRegistry.showArchived.switch.pwLocator.click();
      await firstRow.archivedIcon.pwLocator.waitFor();
    });

    await test.step('Unarchive', async () => {
      await (await firstRow.actions.open()).switchArchived.pwLocator.click();
      await firstRow.archivedIcon.pwLocator.waitFor({ state: 'hidden' });
    });
  });

  test('Move', async ({ backgroundApiWorkspace, newWorkspace, authedPage }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);
    const modelRegistry = workspaceDetails.modelRegistry;
    const firstRow = modelRegistry.table.table.rows.nth(0);

    await test.step('Create destination workspace', async () => {
      const destinationWorkspace = (
        await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new())
      ).workspace;
      workspaces.set('destination', destinationWorkspace);
    });

    await test.step('Move model to destination workspace', async () => {
      await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
      await workspaceDetails.modelRegistryTab.pwLocator.click();

      await (await firstRow.actions.open()).move.pwLocator.click();

      const destinationWorkspaceName = workspaces.get('destination')?.name ?? '';
      await modelRegistry.modelMoveModal.workspaceSelect.pwLocator.fill(destinationWorkspaceName);
      await modelRegistry.modelMoveModal.workspaceSelect.pwLocator.press('Enter');

      await modelRegistry.modelMoveModal.footer.submit.pwLocator.click();

      await expect(modelRegistry.notification.description.pwLocator).toContainText(
        `${modelName} moved to workspace ${workspaces.get('destination')?.name}`,
      );

      await authedPage.reload();
      await modelRegistry.noModelsMessage.pwLocator.waitFor();
    });

    await test.step('Check destination workspace', async () => {
      await workspaceDetails.gotoWorkspace(workspaces.get('destination')?.id);
      await workspaceDetails.modelRegistryTab.pwLocator.click();

      await expect(firstRow.name.pwLocator).toContainText(modelName);
    });
  });
});
