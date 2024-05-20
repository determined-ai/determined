import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { BasePage } from 'e2e/models/BasePage';
import { WorkspaceCreateModal } from 'e2e/models/components/WorkspaceCreateModal';
import { Workspaces } from 'e2e/models/pages/Workspaces';
import { randId, safeName } from 'e2e/utils/naming';

test.describe('Projects', () => {
  test.setTimeout(120_000);
  let wsCreatedWithButton: string = '';
  let wsCreatedWithSidebar: string = '';
  const createWorkspaceAllFields = async function (
    modal: WorkspaceCreateModal,
    wsNamePrefix: string,
  ): Promise<string> {
    const fullName = safeName(wsNamePrefix);
    await modal.workspaceName.pwLocator.fill(fullName);

    await modal.useAgentUser.switch.pwLocator.click();
    await modal.agentUid.pwLocator.fill(randId().toString());
    await modal.agentUser.pwLocator.fill(safeName('user'));

    await modal.useAgentGroup.switch.pwLocator.click();
    await modal.agentGid.pwLocator.fill(randId().toString());
    await modal.agentGroup.pwLocator.fill(safeName('group'));

    await modal.useCheckpointStorage.switch.pwLocator.click();
    await expect(modal.checkpointCodeEditor.pwLocator).toBeVisible();
    await modal.useCheckpointStorage.switch.pwLocator.click();
    await expect(modal.checkpointCodeEditor.pwLocator).not.toBeVisible();

    await modal.footer.submit.pwLocator.click();
    await expect(modal.pwLocator).not.toBeVisible();
    return fullName;
  };

  test.beforeEach(async ({ authedPage }) => {
    await expect(authedPage).toHaveTitle(BasePage.getTitle('Home'));
    await expect(authedPage).toHaveURL(/dashboard/);
  });

  test.afterEach(async ({ authedPage }) => {
    const workspacesPage = new Workspaces(authedPage);
    await test.step('Delete a workspace', async () => {
      if (wsCreatedWithButton !== '') {
        await workspacesPage.nav.sidebar.workspaces.pwLocator.click();
        const workspaceCard = workspacesPage.list.cardWithName(wsCreatedWithButton);
        await (await workspaceCard.actionMenu.open()).delete.pwLocator.click();
        await workspacesPage.deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithButton);
        await workspacesPage.deleteModal.footer.submit.pwLocator.click();
      }
    });
    await test.step('Delete a workspace through sidebar', async () => {
      if (wsCreatedWithSidebar !== '') {
        const workspaceItem = workspacesPage.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithSidebar);
        await (await workspaceItem.actionMenu.open()).delete.pwLocator.click();
        await workspacesPage.deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithButton); // wrong name
        await expect(workspacesPage.deleteModal.footer.submit.pwLocator).toBeDisabled();
        await workspacesPage.deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithSidebar);
        await workspacesPage.deleteModal.footer.submit.pwLocator.click();
      }
    });
  });

  test('Projects and Workspaces CRUD', async ({ authedPage }) => {
    const workspacesPage = new Workspaces(authedPage);

    await test.step('Navigate to Workspaces', async () => {
      await workspacesPage.nav.sidebar.workspaces.pwLocator.click();
      await authedPage.waitForURL(`**/${workspacesPage.url}?**`); // glob pattern for query params
      await expect.soft(authedPage).toHaveTitle(workspacesPage.title);
    });

    await test.step('Create a workspace', async () => {
      await workspacesPage.list.newWorkspaceButton.pwLocator.click();
      wsCreatedWithButton = await createWorkspaceAllFields(
        workspacesPage.createModal,
        'fromButton',
      );

      await expect(
        workspacesPage.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator,
      ).toBeVisible();
      await workspacesPage.nav.sidebar.workspaces.pwLocator.click();
      await expect(workspacesPage.list.cardWithName(wsCreatedWithButton).pwLocator).toBeVisible();
    });
    await test.step('Create a workspace through the sidebar', async () => {
      await workspacesPage.nav.sidebar.workspaces.pwLocator.hover();
      await workspacesPage.nav.sidebar.createWorkspace.pwLocator.click();
      wsCreatedWithSidebar = await createWorkspaceAllFields(
        workspacesPage.createModal,
        'fromSidebar',
      );

      await expect(
        workspacesPage.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithSidebar).pwLocator,
      ).toBeVisible();
      await workspacesPage.nav.sidebar.workspaces.pwLocator.click();
      await expect(workspacesPage.list.cardWithName(wsCreatedWithSidebar).pwLocator).toBeVisible();
    });

    await test.step('Create projects', async () => {});
    await test.step('Archive a project', async () => {});
    await test.step('Unarchive a project', async () => {});
    await test.step('Navigation on projects page - sorting and list', async () => {});
    await test.step('Create a model with all possible metadata', async () => {});
    await test.step('Archive a model', async () => {});
    await test.step('Unarchive a model', async () => {});
    await test.step('Move a model between projects', async () => {});
    await test.step('Launch JupyterLab, kill the task, view logs', async () => {});
    await test.step('Navigate with the breadcrumb and workspace page', async () => {});
    await test.step('Navigation on workspace page', async () => {});
    await test.step('Navigation to wokspace on the sidebar', async () => {});
    await test.step('Edit a workspace through workspaces page', async () => {});
    await test.step('Edit a workspace through the sidebar', async () => {});
    await test.step('Archive a workspace', async () => {});
    await test.step('Unarchive a workspace', async () => {});
    await test.step('Unpin a workspace through the sidebar', async () => {});
    await test.step('Pin a workspace through the sidebar', async () => {});
    await test.step('Delete a model', async () => {});
    await test.step('Delete a project', async () => {});
  });
});
