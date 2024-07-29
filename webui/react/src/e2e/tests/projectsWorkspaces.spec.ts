import { expect, test } from 'e2e/fixtures/global-fixtures';
import { WorkspaceCreateModal } from 'e2e/models/components/WorkspaceCreateModal';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';
import { WorkspaceList } from 'e2e/models/pages/WorkspaceList';
import { randId, safeName } from 'e2e/utils/naming';

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
  await modal.checkpointCodeEditor.pwLocator.waitFor();
  await modal.useCheckpointStorage.switch.pwLocator.click();
  await modal.checkpointCodeEditor.pwLocator.waitFor({ state: 'hidden' });

  await modal.footer.submit.pwLocator.click();
  await modal.pwLocator.waitFor({ state: 'hidden' });
  return fullName;
};

test.describe('Workspace UI CRUD', () => {
  const workspaceIds: number[] = [];

  test.beforeEach(async ({ authedPage }) => {
    const workspaceList = new WorkspaceList(authedPage);
    await workspaceList.goto();
    // wait for this because the showArchived switch defaults to false until the page fully loads
    await workspaceList.workspaceCards.pwLocator
      .nth(0)
      .or(workspaceList.noWorkspacesMessage.pwLocator)
      .or(workspaceList.noMatchingWorkspacesMessage.pwLocator)
      .waitFor();
    await workspaceList.showArchived.switch.uncheck();
  });

  test.afterAll(async ({ backgroundApiWorkspace }) => {
    for (const workspace of workspaceIds) {
      await backgroundApiWorkspace.deleteWorkspace(workspace);
    }
  });

  test('Workspace with Button', async ({ authedPage }) => {
    let wsCreatedWithButton = '';
    const workspaceList = new WorkspaceList(authedPage);
    const sidebar = workspaceList.nav.sidebar;
    await test.step('Create a Workspace from Button', async () => {
      await workspaceList.newWorkspaceButton.pwLocator.click();
      wsCreatedWithButton = await createWorkspaceAllFields(workspaceList.createModal, 'fromButton');
      await sidebar.workspaces.pwLocator.click();
      await sidebar.sidebarWorkspaceItem(wsCreatedWithButton).pwLocator.waitFor();
    });

    await test.step('Get Workspace ID', async () => {
      const workspaceDetails = new WorkspaceDetails(authedPage);
      await workspaceList.cardByName(wsCreatedWithButton).pwLocator.click();
      workspaceIds.push(await workspaceDetails.getIdFromUrl());
      await authedPage.goBack();
    });

    await test.step('Delete Workspace from Card', async () => {
      const deleteModal = workspaceList.deleteModal;
      if (wsCreatedWithButton !== '') {
        await workspaceList.nav.sidebar.workspaces.pwLocator.click();
        const workspaceCard = workspaceList.cardByName(wsCreatedWithButton);
        await (await workspaceCard.actionMenu.open()).delete.pwLocator.click();
        await deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithButton);
        await deleteModal.footer.submit.pwLocator.click();
      }
    });
  });

  test('Workspace with Sidebar', async ({ authedPage }) => {
    let wsCreatedWithSidebar = '';
    const workspaceList = new WorkspaceList(authedPage);
    const sidebar = workspaceList.nav.sidebar;

    await test.step('Create a Workspace from Sidebar', async () => {
      await sidebar.workspaces.pwLocator.hover();
      await sidebar.createWorkspaceFromHover.pwLocator.click();
      wsCreatedWithSidebar = await createWorkspaceAllFields(
        workspaceList.createModal,
        'fromSidebar',
      );
      await sidebar.workspaces.pwLocator.click();
      await sidebar.sidebarWorkspaceItem(wsCreatedWithSidebar).pwLocator.waitFor();
      await workspaceList.cardByName(wsCreatedWithSidebar).pwLocator.click();
    });

    await test.step('Get Workspace ID', async () => {
      const workspaceDetails = new WorkspaceDetails(authedPage);
      workspaceIds.push(await workspaceDetails.getIdFromUrl());
      await authedPage.goBack();
    });

    await test.step('Delete Workspace from Sidebar', async () => {
      const deleteModal = workspaceList.deleteModal;
      if (wsCreatedWithSidebar !== '') {
        const workspaceItem = workspaceList.nav.sidebar.sidebarWorkspaceItem(wsCreatedWithSidebar);
        await (await workspaceItem.actionMenu.open()).delete.pwLocator.click();
        await deleteModal.nameConfirmation.pwLocator.fill(wsCreatedWithSidebar);
        await deleteModal.footer.submit.pwLocator.click();
      }
    });
  });

  test('Pin and Unpin a Workspace from Card', async ({ authedPage, backgroundApiWorkspace }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const newWorkspace = await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new());
    workspaceIds.push(newWorkspace.workspace.id!);
    const workspaceCard = workspaceList.cardByName(newWorkspace.workspace.name!);
    const sidebarItem = workspaceList.nav.sidebar.sidebarWorkspaceItem(
      newWorkspace.workspace.name!,
    );
    const pinMenuItem = workspaceCard.actionMenu.pin;

    await test.step('Unpin', async () => {
      await authedPage.reload();
      await workspaceCard.actionMenu.open();
      await expect(pinMenuItem.pwLocator).toHaveText('Unpin from sidebar');
      await pinMenuItem.pwLocator.click();
      await sidebarItem.pwLocator.waitFor({ state: 'hidden' });
    });
    await test.step('Pin', async () => {
      await workspaceCard.actionMenu.open();
      await expect(pinMenuItem.pwLocator).toHaveText('Pin to sidebar');
      await pinMenuItem.pwLocator.click();
      await sidebarItem.pwLocator.waitFor();
    });
  });

  test('Unpin a Workspace from Sidebar', async ({ authedPage, backgroundApiWorkspace }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const newWorkspace = await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new());
    workspaceIds.push(newWorkspace.workspace.id!);
    const workspaceCard = workspaceList.cardByName(newWorkspace.workspace.name!);
    const sidebarItem = workspaceList.nav.sidebar.sidebarWorkspaceItem(
      newWorkspace.workspace.name!,
    );

    await authedPage.reload();
    await sidebarItem.pwLocator.waitFor({ timeout: 10_000 });
    await (await sidebarItem.actionMenu.open()).pin.pwLocator.click();
    await sidebarItem.pwLocator.waitFor({ state: 'hidden' });
    await expect((await workspaceCard.actionMenu.open()).pin.pwLocator).toHaveText(
      'Pin to sidebar',
    );
  });

  test('Archive and Unarchive Workspace from Card', async ({
    authedPage,
    backgroundApiWorkspace,
  }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const newWorkspace = await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new());
    workspaceIds.push(newWorkspace.workspace.id!);
    const workspaceCard = workspaceList.cardByName(newWorkspace.workspace.name!);
    const sidebarItem = workspaceList.nav.sidebar.sidebarWorkspaceItem(
      newWorkspace.workspace.name!,
    );
    const archiveMenuItem = workspaceCard.actionMenu.archive;

    await test.step('Archive', async () => {
      await authedPage.reload();
      await workspaceCard.actionMenu.open();
      await expect(archiveMenuItem.pwLocator).toHaveText('Archive');
      await archiveMenuItem.pwLocator.click();
      await workspaceCard.pwLocator.waitFor({ state: 'hidden' });
      await sidebarItem.pwLocator.waitFor();
    });

    await test.step('Unarchive', async () => {
      await workspaceList.showArchived.switch.pwLocator.click();
      await workspaceCard.archivedBadge.pwLocator.waitFor();
      await workspaceCard.actionMenu.open();
      await expect(archiveMenuItem.pwLocator).toHaveText('Unarchive');
      await archiveMenuItem.pwLocator.click();
      await workspaceCard.archivedBadge.pwLocator.waitFor({ state: 'hidden' });
    });
  });

  test('Archive and Unarchive Workspace from Sidebar', async ({
    authedPage,
    backgroundApiWorkspace,
  }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const newWorkspace = await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new());
    workspaceIds.push(newWorkspace.workspace.id!);
    const workspaceCard = workspaceList.cardByName(newWorkspace.workspace.name!);
    const sidebarItem = workspaceList.nav.sidebar.sidebarWorkspaceItem(
      newWorkspace.workspace.name!,
    );
    await authedPage.reload();
    const archiveMenuItem = sidebarItem.actionMenu.archive;

    await test.step('Archive', async () => {
      await sidebarItem.actionMenu.open();
      await expect(archiveMenuItem.pwLocator).toHaveText('Archive');
      await archiveMenuItem.pwLocator.click();
      await workspaceCard.pwLocator.waitFor({ state: 'hidden', timeout: 10_000 });
    });

    await test.step('Unarchive', async () => {
      await sidebarItem.actionMenu.open();
      await expect(archiveMenuItem.pwLocator).toHaveText('Unarchive');
      await archiveMenuItem.pwLocator.click();
      await workspaceCard.pwLocator.waitFor({ timeout: 10_000 });
    });
  });

  test('Edit a Workspace', async ({ authedPage, backgroundApiWorkspace }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const newWorkspace = await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new());
    workspaceIds.push(newWorkspace.workspace.id!);
    const workspaceCard = workspaceList.cardByName(newWorkspace.workspace.name!);

    await authedPage.reload();
    await (await workspaceCard.actionMenu.open()).edit.pwLocator.click();
    const newName = await createWorkspaceAllFields(workspaceList.createModal, 'editedWorkspace');
    const workspaceCardEdited = workspaceList.cardByName(newName);
    await Promise.all([
      workspaceCard.pwLocator.waitFor({ state: 'hidden', timeout: 10_000 }),
      workspaceCardEdited.pwLocator.waitFor({ timeout: 10_000 }),
    ]);
  });
});

test.describe('With a Workspace', () => {
  test.beforeEach(async ({ authedPage, newWorkspace }) => {
    const workspaceList = new WorkspaceList(authedPage);
    const workspaceCard = workspaceList.cardByName(newWorkspace.response.workspace.name);

    await test.step('Navigate to Workspaces', async () => {
      await workspaceList.goto();
      await workspaceCard.pwLocator.waitFor({ timeout: 10_000 });
    });
  });

  test('Attempt to delete a workspace but with bad validation', async ({
    authedPage,
    newWorkspace,
  }) => {
    const workspaceList = new WorkspaceList(authedPage);
    const deleteModal = workspaceList.deleteModal;
    const workspaceCard = workspaceList.cardByName(newWorkspace.response.workspace.name);

    await workspaceList.nav.sidebar.workspaces.pwLocator.click();
    await (await workspaceCard.actionMenu.open()).delete.pwLocator.click();
    await deleteModal.nameConfirmation.pwLocator.fill('bad validation');
    await expect(deleteModal.footer.submit.pwLocator).toBeDisabled();
  });

  test.describe('Project UI CRUD', () => {
    const projectIds: number[] = [];

    test.beforeEach(async ({ authedPage, newWorkspace }) => {
      const workspaceDetails = new WorkspaceDetails(authedPage);
      await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
      await workspaceDetails.workspaceProjects.showArchived.switch.uncheck();
    });

    test.afterAll(async ({ backgroundApiProject }) => {
      for (const project of projectIds) {
        await backgroundApiProject.deleteProject(project);
      }
    });

    test('Create a Project', async ({ authedPage, newWorkspace }) => {
      const projectName = safeName('test-project');
      const workspaceList = new WorkspaceList(authedPage);
      const workspaceDetails = new WorkspaceDetails(authedPage);
      const projectDetails = new ProjectDetails(authedPage);

      const sidebar = workspaceList.nav.sidebar;
      const projects = workspaceDetails.workspaceProjects;

      await test.step('Create a Project', async () => {
        await projects.newProject.pwLocator.click();
        await projects.createModal.projectName.pwLocator.fill(projectName);
        await projects.createModal.description.pwLocator.fill(randId());
        await projects.createModal.footer.submit.pwLocator.click();
        projectIds.push(await projectDetails.getIdFromUrl());
        await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
        await projects.cardByName(projectName).pwLocator.waitFor();
      });

      await test.step('Delete a Project', async () => {
        await sidebar.sidebarWorkspaceItem(newWorkspace.response.workspace.name).pwLocator.click();
        await workspaceDetails.projectsTab.pwLocator.click();
        const projectCard = projects.cardByName(projectName);
        await projectCard.actionMenu.open();
        await projectCard.actionMenu.delete.pwLocator.click();
        await projects.deleteModal.nameConfirmation.pwLocator.fill(projectName);
        await projects.deleteModal.footer.submit.pwLocator.click();
      });
    });

    test('Archive and Unarchive Project', async ({
      authedPage,
      newWorkspace,
      backgroundApiProject,
    }) => {
      const workspaceDetails = new WorkspaceDetails(authedPage);

      const newProject = await backgroundApiProject.createProject(
        newWorkspace.response.workspace.id,
        backgroundApiProject.new(),
      );
      projectIds.push(newProject.project.id);
      const projectCard = workspaceDetails.workspaceProjects.cardByName(newProject.project.name);
      const archiveMenuItem = projectCard.actionMenu.archive;

      await test.step('Archive', async () => {
        await authedPage.reload();
        await projectCard.actionMenu.open();
        await expect(archiveMenuItem.pwLocator).toHaveText('Archive');
        await archiveMenuItem.pwLocator.click();
        await projectCard.pwLocator.waitFor({ state: 'hidden' });
      });

      await test.step('Unarchive', async () => {
        await workspaceDetails.workspaceProjects.showArchived.switch.pwLocator.click();
        await projectCard.archivedBadge.pwLocator.waitFor();
        await projectCard.actionMenu.open();
        await expect(archiveMenuItem.pwLocator).toHaveText('Unarchive');
        await archiveMenuItem.pwLocator.click();
        await projectCard.archivedBadge.pwLocator.waitFor({ state: 'hidden' });
      });
    });

    // remianing tests
    // test('Navigation on Projects Page - Sorting and List', async () => {});
    // test('Navigation on Workspaces Page - Sorting and List', async () => {});
    // test('Navigate with Breadcrumbs on the Workspaces Page', async () => {});
    // test('Navigate with Breadcrumbs on the Projects Page', async () => {});
    // test.describe('With Model Teardown', () => {
    //   test('Use UI to create and Delete a Model with All Possible Metadata', async () => {});
    //   test('Create a model with backend, Archive and Unarchive Model', async () => {});
    //   test('Move a Model Between Projects', async () => {});
    // });
    // test.describe('Task', () => {
    //   beforeAll('Visit tasks page')
    //   test('Launch JupyterLab, View Task Logs', async () => {});
    //   test('Launch JupyterLab, Kill, View Task Logs', async () => {});
    //   afterAll('Kill Tasks')
    // });
  });
});
