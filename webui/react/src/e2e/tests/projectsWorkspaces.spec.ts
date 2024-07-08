import { AuthFixture } from 'e2e/fixtures/auth.fixture';
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
    await modal.agentUid.pwLocator.fill(randId());
    await modal.agentUser.pwLocator.fill(safeName('user'));

    await modal.useAgentGroup.switch.pwLocator.click();
    await modal.agentGid.pwLocator.fill(randId());
    await modal.agentGroup.pwLocator.fill(safeName('group'));

    await modal.useCheckpointStorage.switch.pwLocator.click();
    await modal.useCheckpointStorage.switch.pwLocator.click();
    await modal.checkpointCodeEditor.pwLocator.waitFor({ state: 'hidden' });

    await modal.footer.submit.pwLocator.click();
    await modal.pwLocator.waitFor({ state: 'hidden' });
    return fullName;
  };

  test.beforeEach(async ({ authedPage }) => {
    const workspaceList = new WorkspaceList(authedPage);

    await test.step('Navigate to Workspaces', async () => {
      await workspaceList.nav.sidebar.workspaces.pwLocator.click();
      await authedPage.waitForURL(`**/${workspaceList.url}?**`); // glob pattern for query params
      await expect(authedPage).toHaveDeterminedTitle(workspaceList.title);
    });
  });

  test.beforeAll(async ({ browser, dev }) => {
    const pageSetup = await browser.newPage();
    await dev.setServerAddress(pageSetup);
    const authFixtureSetup = new AuthFixture(pageSetup);
    await authFixtureSetup.login();
    const workspaceListSetup = new WorkspaceList(pageSetup);
    await workspaceListSetup.goto();
    const sidebar = workspaceListSetup.nav.sidebar;
    await test.step('Create a Workspace from Button', async () => {
      await workspaceListSetup.newWorkspaceButton.pwLocator.click();
      wsCreatedWithButton = await createWorkspaceAllFields(
        workspaceListSetup.createModal,
        'fromButton',
      );

      await sidebar.workspaces.pwLocator.click();
      await sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.waitFor();
      await workspaceListSetup.cardByName(wsCreatedWithButton).pwLocator.waitFor();
    });
    await test.step('Create a Workspace from Sidebar', async () => {
      await sidebar.workspaces.pwLocator.hover();
      await sidebar.createWorkspaceFromHover.pwLocator.click();
      wsCreatedWithSidebar = await createWorkspaceAllFields(
        workspaceListSetup.createModal,
        'fromSidebar',
      );
      await sidebar.workspaces.pwLocator.click();
      await sidebar.sidebarWorkspaceItem(wsCreatedWithSidebar).pwLocator.waitFor();
      await workspaceListSetup.cardByName(wsCreatedWithSidebar).pwLocator.waitFor();
    });
    await test.step('Create a Project', async () => {
      const workspaceDetailsSetup = new WorkspaceDetails(pageSetup);
      await sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.click();
      await workspaceDetailsSetup.projectsTab.pwLocator.click();
      const projects = workspaceDetailsSetup.workspaceProjects;
      await projects.newProject.pwLocator.click();
      projectOneName = `test-1-${randId()}`;
      await projects.createModal.projectName.pwLocator.fill(projectOneName);
      await projects.createModal.description.pwLocator.fill(randId());
      await projects.createModal.footer.submit.pwLocator.click();
      await pageSetup.waitForURL('**/projects/*/experiments');
      await sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.click();
      await projects.cardByName(projectOneName).pwLocator.waitFor();
    });
    await authFixtureSetup.logout();
    await pageSetup.close();
  });

  test.afterAll(async ({ browser, dev }) => {
    const pageTeardown = await browser.newPage();
    await dev.setServerAddress(pageTeardown);

    const authFixtureSetup = new AuthFixture(pageTeardown);
    await authFixtureSetup.login();
    const workspaceListTeardown = new WorkspaceList(pageTeardown);
    await workspaceListTeardown.goto();
    const deleteModal = workspaceListTeardown.deleteModal;

    await test.step('Delete a Project', async () => {
      const workspaceDetailsTeardown = new WorkspaceDetails(pageTeardown);
      const projects = workspaceDetailsTeardown.workspaceProjects;
      await workspaceListTeardown.nav.sidebar
        .sidebarWorkspaceItem(wsCreatedWithButton)
        .pwLocator.click();
      await workspaceDetailsTeardown.projectsTab.pwLocator.click();
      const projectCard = projects.cardByName(projectOneName);
      await projectCard.actionMenu.open();
      await projectCard.actionMenu.delete.pwLocator.click();
      await projects.deleteModal.nameConfirmation.pwLocator.fill(projectOneName);
      await projects.deleteModal.footer.submit.pwLocator.click();
    });
    await test.step('Delete Workspace from Card', async () => {
      if (wsCreatedWithButton !== '') {
        await workspaceListTeardown.nav.sidebar.workspaces.pwLocator.click();
        const workspaceCard = workspaceListTeardown.cardByName(wsCreatedWithButton);
        await (await workspaceCard.actionMenu.open()).delete.pwLocator.click();
        await deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithButton);
        await deleteModal.footer.submit.pwLocator.click();
      }
    });
    await test.step('Delete Workspace from Sidebar', async () => {
      if (wsCreatedWithSidebar !== '') {
        const workspaceItem =
          workspaceListTeardown.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithSidebar);
        await (await workspaceItem.actionMenu.open()).delete.pwLocator.click();
        await deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithSidebar);
        await deleteModal.footer.submit.pwLocator.click();
      }
    });
    await authFixtureSetup.logout();
    await pageTeardown.close();
  });

  test('Attempt to delete a workspace but with bad validation', async ({ authedPage }) => {
    const workspaceList = new WorkspaceList(authedPage);
    const deleteModal = workspaceList.deleteModal;

    await workspaceList.nav.sidebar.workspaces.pwLocator.click();
    const workspaceCard = workspaceList.cardByName(wsCreatedWithButton);
    await (await workspaceCard.actionMenu.open()).delete.pwLocator.click();
    await deleteModal.nameConfirmation.pwLocator.fill('bad validation');
    await expect(deleteModal.footer.submit.pwLocator).toBeDisabled();
  });

  // test('Projects and Workspaces archival and pinning', async ({ authedPage }) => {
  //   await test.step('Archive a workspace', async () => {});
  //   await test.step('Unarchive a workspace', async () => {});
  //   await test.step('Unpin a workspace through the sidebar', async () => {});
  //   await test.step('Pin a workspace through the sidebar', async () => {});
  //   await test.step('Archive a project', async () => {});
  //   await test.step('Unarchive a project', async () => {});
  // })

  // test('Projects and Workspaces CRUD', async ({ authedPage }) => {
  //   const workspaceList = new WorkspaceList(authedPage);
  //   const workspaceDetails = new WorkspaceDetails(authedPage);
  //   await test.step('Navigation on Projects Page - Sorting and List', async () => {});
  //   await test.step('Create a Model with All Possible Metadata', async () => {});
  //   await test.step('Archive a Model', async () => {});
  //   await test.step('Unarchive a Model', async () => {});
  //   await test.step('Move a Model Between Projects', async () => {});
  //   await test.step('Launch JupyterLab, Kill the Task, View Logs', async () => {});
  //   await test.step('Navigate with the Breadcrumb and Workspace Page', async () => {});
  //   await test.step('Navigation on Workspace Page', async () => {});
  //   await test.step('Edit a Workspace', async () => {});
  //   await test.step('Delete a Model', async () => {});
  // });
});
