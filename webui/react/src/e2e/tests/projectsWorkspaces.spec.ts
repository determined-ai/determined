import { expect } from '@playwright/test';
import { v4 } from 'uuid';

import { test } from 'e2e/fixtures/global-fixtures';
import { BasePage } from 'e2e/models/common/base/BasePage';
import { WorkspaceCreateModal } from 'e2e/models/components/WorkspaceCreateModal';
import { Workspaces } from 'e2e/models/pages/Workspaces';
import { randId, safeName } from 'e2e/utils/naming';

test.describe('Projects', () => {
  test.slow();
  let wsCreatedWithButton = '';
  let wsCreatedWithSidebar = '';
  let projectOneName = '';

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
    const workspacesPage = new Workspaces(authedPage);

    await expect(authedPage).toHaveTitle(BasePage.getTitle('Home'));
    await expect(authedPage).toHaveURL(/dashboard/);
    await test.step('Navigate to Workspaces', async () => {
      await workspacesPage.nav.sidebar.workspaces.pwLocator.click();
      await authedPage.waitForURL(`**/${workspacesPage.url}?**`); // glob pattern for query params
      await expect.soft(authedPage).toHaveTitle(workspacesPage.title);
    });
  });

  test.afterEach(async ({ authedPage }) => {
    const workspacesPage = new Workspaces(authedPage);
    await test.step('Delete Workspace from Card', async () => {
      if (wsCreatedWithButton !== '') {
        await workspacesPage.nav.sidebar.workspaces.pwLocator.click();
        const workspaceCard = workspacesPage.list.cardWithName(wsCreatedWithButton);
        await (await workspaceCard.actionMenu.open()).delete.pwLocator.click();
        await workspacesPage.deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithButton);
        await workspacesPage.deleteModal.footer.submit.pwLocator.click();
      }
    });
    await test.step('Delete Workspace from Sidebar', async () => {
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

  // test('Projects and Workspaces archival and pinning', async ({ page }) => {
  //   await test.step('Archive a workspace', async () => {});
  //   await test.step('Unarchive a workspace', async () => {});
  //   await test.step('Unpin a workspace through the sidebar', async () => {});
  //   await test.step('Pin a workspace through the sidebar', async () => {});
  //   await test.step('Archive a project', async () => {});
  //   await test.step('Unarchive a project', async () => {});
  // })

  test('Projects and Workspaces CRUD', async ({ authedPage }) => {
    const workspacesPage = new Workspaces(authedPage);

    await test.step('Create a Workspace from Card', async () => {
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
    await test.step('Create a Workspace from Sidebar', async () => {
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

    await test.step('Create Projects', async () => {
      await workspacesPage.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.click();
      const projects = workspacesPage.details.projects;
      await projects.pwLocator.click();
      await projects.newProject.pwLocator.click();
      projectOneName = `test-1-${v4()}`;
      await projects.createModal.projectName.pwLocator.fill(projectOneName);
      await projects.createModal.description.pwLocator.fill(v4());
      await projects.createModal.footer.submit.pwLocator.click();
      await authedPage.waitForURL('**/projects/*/experiments');
      await workspacesPage.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.click();
      await expect(projects.cardWithName(projectOneName).pwLocator).toBeVisible();
    });

    await test.step('Navigation on Projects Page - Sorting and List', async () => {});
    await test.step('Create a Model with All Possible Metadata', async () => {});
    await test.step('Archive a Model', async () => {});
    await test.step('Unarchive a Model', async () => {});
    await test.step('Move a Model Between Projects', async () => {});
    await test.step('Launch JupyterLab, Kill the Task, View Logs', async () => {});
    await test.step('Navigate with the Breadcrumb and Workspace Page', async () => {});
    await test.step('Navigation on Workspace Page', async () => {});
    await test.step('Edit a Workspace', async () => {});
    await test.step('Delete a Model', async () => {});
    await test.step('Delete a Project', async () => {
      await workspacesPage.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.click();
      await workspacesPage.details.projects.pwLocator.click();
      const projectContent = workspacesPage.details.projects;
      const projectCard = projectContent.cardWithName(projectOneName);
      await projectCard.actionMenu.open();
      await projectCard.actionMenu.delete.pwLocator.click();
      await projectContent.deleteModal.nameConfirmation.pwLocator.fill(projectOneName);
      await projectContent.deleteModal.footer.submit.pwLocator.click();
    });
  });
});
