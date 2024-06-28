import { v4 } from 'uuid';

import { expect, test } from 'e2e/fixtures/global-fixtures';
import { WorkspaceCreateModal } from 'e2e/models/components/WorkspaceCreateModal';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';
import { WorkspaceList } from 'e2e/models/pages/WorkspaceList';
import { randId, safeName } from 'e2e/utils/naming';

test.describe('Projects', () => {
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
    const workspaceList = new WorkspaceList(authedPage);

    await expect(authedPage).toHaveDeterminedTitle('Home');
    await expect(authedPage).toHaveURL(/dashboard/);
    await test.step('Navigate to Workspaces', async () => {
      await workspaceList.nav.sidebar.workspaces.pwLocator.click();
      await authedPage.waitForURL(`**/${workspaceList.url}?**`); // glob pattern for query params
      await expect.soft(authedPage).toHaveDeterminedTitle(workspaceList.title);
    });
  });

  test.afterEach(async ({ authedPage }) => {
    const workspaceList = new WorkspaceList(authedPage);
    await test.step('Delete Workspace from Card', async () => {
      if (wsCreatedWithButton !== '') {
        await workspaceList.nav.sidebar.workspaces.pwLocator.click();
        const workspaceCard = workspaceList.cardByName(wsCreatedWithButton);
        await (await workspaceCard.actionMenu.open()).delete.pwLocator.click();
        await workspaceList.deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithButton);
        await workspaceList.deleteModal.footer.submit.pwLocator.click();
      }
    });
    await test.step('Delete Workspace from Sidebar', async () => {
      if (wsCreatedWithSidebar !== '') {
        const workspaceItem = workspaceList.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithSidebar);
        await (await workspaceItem.actionMenu.open()).delete.pwLocator.click();
        await workspaceList.deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithButton); // wrong name
        await expect(workspaceList.deleteModal.footer.submit.pwLocator).toBeDisabled();
        await workspaceList.deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithSidebar);
        await workspaceList.deleteModal.footer.submit.pwLocator.click();
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
    const workspaceList = new WorkspaceList(authedPage);
    const workspaceDetails = new WorkspaceDetails(authedPage);

    await test.step('Create a Workspace from Card', async () => {
      await workspaceList.newWorkspaceButton.pwLocator.click();
      wsCreatedWithButton = await createWorkspaceAllFields(workspaceList.createModal, 'fromButton');

      await expect(
        workspaceList.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator,
      ).toBeVisible();
      await workspaceList.nav.sidebar.workspaces.pwLocator.click();
      await workspaceList.cardByName(wsCreatedWithButton).pwLocator.waitFor();
    });
    await test.step('Create a Workspace from Sidebar', async () => {
      await workspaceList.nav.sidebar.workspaces.pwLocator.hover();
      await workspaceList.nav.sidebar.createWorkspaceFromHover.pwLocator.click();
      wsCreatedWithSidebar = await createWorkspaceAllFields(
        workspaceList.createModal,
        'fromSidebar',
      );

      await expect(
        workspaceList.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithSidebar).pwLocator,
      ).toBeVisible();
      await workspaceList.nav.sidebar.workspaces.pwLocator.click();
      await expect(workspaceList.cardByName(wsCreatedWithSidebar).pwLocator).toBeVisible();
    });

    await test.step('Create Projects', async () => {
      await workspaceList.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.click();
      await workspaceDetails.projectsTab.pwLocator.click();
      const projects = workspaceDetails.workspaceProjects;
      await projects.newProject.pwLocator.click();
      projectOneName = `test-1-${v4()}`;
      await projects.createModal.projectName.pwLocator.fill(projectOneName);
      await projects.createModal.description.pwLocator.fill(v4());
      await projects.createModal.footer.submit.pwLocator.click();
      await authedPage.waitForURL('**/projects/*/experiments');
      await workspaceList.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.click();
      await expect(projects.cardByName(projectOneName).pwLocator).toBeVisible();
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
      const projects = workspaceDetails.workspaceProjects;
      await workspaceList.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.click();
      await workspaceDetails.projectsTab.pwLocator.click();
      const projectCard = projects.cardByName(projectOneName);
      await projectCard.actionMenu.open();
      await projectCard.actionMenu.delete.pwLocator.click();
      await projects.deleteModal.nameConfirmation.pwLocator.fill(projectOneName);
      await projects.deleteModal.footer.submit.pwLocator.click();
    });
  });
});
