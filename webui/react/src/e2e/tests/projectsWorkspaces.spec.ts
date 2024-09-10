import _ from 'lodash';

import { expect, test } from 'e2e/fixtures/global-fixtures';
import { WorkspaceCreateModal } from 'e2e/models/components/WorkspaceCreateModal';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';
import { WorkspaceList } from 'e2e/models/pages/WorkspaceList';
import { randId, safeName } from 'e2e/utils/naming';
import { V1Workspace } from 'services/api-ts-sdk';

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

const getCurrentWorkspaceNames = async (workspaceList: WorkspaceList) => {
  await workspaceList.workspaceCards.pwLocator.nth(0).waitFor();

  const cardTitles = await workspaceList.workspaceCards.title.pwLocator.all();
  return await Promise.all(
    cardTitles.map(async (title) => {
      return await title.textContent();
    }),
  );
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

  test('Pin and Unpin a Workspace from Card', async ({ authedPage, apiWorkspace }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const newWorkspace = await apiWorkspace.createWorkspace(apiWorkspace.new());
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

  test('Unpin a Workspace from Sidebar', async ({ authedPage, apiWorkspace }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const newWorkspace = await apiWorkspace.createWorkspace(apiWorkspace.new());
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

  test('Archive and Unarchive Workspace from Card', async ({ authedPage, apiWorkspace }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const newWorkspace = await apiWorkspace.createWorkspace(apiWorkspace.new());
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

  test('Archive and Unarchive Workspace from Sidebar', async ({ authedPage, apiWorkspace }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const newWorkspace = await apiWorkspace.createWorkspace(apiWorkspace.new());
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

  test('Attempt to delete a workspace but with bad validation', async ({
    authedPage,
    backgroundApiWorkspace,
  }) => {
    const workspaceList = new WorkspaceList(authedPage);
    const deleteModal = workspaceList.deleteModal;

    const newWorkspace = await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new());
    workspaceIds.push(newWorkspace.workspace.id!);
    const workspaceCard = workspaceList.cardByName(newWorkspace.workspace.name!);

    await authedPage.reload();
    await workspaceList.nav.sidebar.workspaces.pwLocator.click();
    await (await workspaceCard.actionMenu.open()).delete.pwLocator.click();
    await deleteModal.nameConfirmation.pwLocator.fill('bad validation');
    await expect(deleteModal.footer.submit.pwLocator).toBeDisabled();
  });
});

test.describe('Workspace List', () => {
  const workspaces: V1Workspace[] = [];

  test.beforeAll(async ({ backgroundApiWorkspace }) => {
    const olderWorkspace = await backgroundApiWorkspace.createWorkspace(
      backgroundApiWorkspace.new({
        // older workspace with first alphabetical name
        workspacePrefix: 'a-test-workspace',
      }),
    );
    workspaces.push(olderWorkspace.workspace);

    const newerWorkspace = await backgroundApiWorkspace.createWorkspace(
      backgroundApiWorkspace.new({
        // newer workspace with last alphabetical name
        workspacePrefix: 'b-test-workspace',
      }),
    );
    workspaces.push(newerWorkspace.workspace);
  });

  test.beforeEach(async ({ authedPage }) => {
    const workspaceList = new WorkspaceList(authedPage);
    await workspaceList.goto();
    await workspaceList.whoseSelect.selectMenuOption('All Workspaces');
    await workspaceList.sortSelect.selectMenuOption('Newest to Oldest');
    await workspaceList.gridListRadioGroup.grid.pwLocator.click();
  });

  test.afterAll(async ({ backgroundApiWorkspace }) => {
    for (const workspace of workspaces) {
      await backgroundApiWorkspace.deleteWorkspace(workspace.id);
    }
  });

  test('Sort', async ({ authedPage }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const namesAfterNewest = await getCurrentWorkspaceNames(workspaceList);
    const idSortedWorkspaceNames = _.orderBy(workspaces, 'id', 'desc').map((w) => w.name);
    expect(idSortedWorkspaceNames).toEqual(
      namesAfterNewest.filter((n) => {
        return n && workspaces.map((w) => w.name).includes(n);
      }),
    );

    await workspaceList.sortSelect.selectMenuOption('Alphabetical');

    const namesAfterAlphabetical = await getCurrentWorkspaceNames(workspaceList);
    const nameSortedWorkspaceNames = _.orderBy(workspaces, 'name', 'asc').map((w) => w.name);
    expect(nameSortedWorkspaceNames).toEqual(
      namesAfterAlphabetical.filter((n) => {
        return n && workspaces.map((w) => w.name).includes(n);
      }),
    );
  });

  test('Filter', async ({ authedPage, apiWorkspace }) => {
    const workspaceList = new WorkspaceList(authedPage);

    const currentUserWorkspace = (await apiWorkspace.createWorkspace(apiWorkspace.new())).workspace;

    const currentUserWorkspaceName = currentUserWorkspace.name;
    const otherUserWorkspaceName = workspaces.map((w) => w.name)[0];

    await authedPage.reload();

    const namesAfterAll = await getCurrentWorkspaceNames(workspaceList);
    expect(namesAfterAll).toContain(otherUserWorkspaceName);
    expect(namesAfterAll).toContain(currentUserWorkspaceName);

    await workspaceList.whoseSelect.selectMenuOption("Others' Workspaces");
    const namesAfterOthers = await getCurrentWorkspaceNames(workspaceList);
    expect(namesAfterOthers).toContain(otherUserWorkspaceName);
    expect(namesAfterOthers).not.toContain(currentUserWorkspaceName);

    await workspaceList.whoseSelect.selectMenuOption('My Workspaces');
    const namesAfterMy = await getCurrentWorkspaceNames(workspaceList);
    expect(namesAfterMy).toContain(currentUserWorkspaceName);
    expect(namesAfterMy).not.toContain(otherUserWorkspaceName);

    await apiWorkspace.deleteWorkspace(currentUserWorkspace.id);
  });

  test('View Toggle', async ({ authedPage }) => {
    const workspaceList = new WorkspaceList(authedPage);

    await workspaceList.gridListRadioGroup.list.pwLocator.click();

    const idSortedWorkspaceNames = _.orderBy(workspaces, 'id', 'desc').map((w) => w.name);

    expect(await workspaceList.table.table.rows.nth(0).name.pwLocator.textContent()).toEqual(
      idSortedWorkspaceNames[0],
    );
  });
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
    const workspaceDetails = new WorkspaceDetails(authedPage);
    const projectDetails = new ProjectDetails(authedPage);

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
      await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
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

  test('Move a Project', async ({
    authedPage,
    newWorkspace,
    backgroundApiWorkspace,
    backgroundApiProject,
  }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);

    const destinationWorkspace = (
      await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new())
    ).workspace;

    const newProject = await backgroundApiProject.createProject(
      newWorkspace.response.workspace.id,
      backgroundApiProject.new(),
    );
    projectIds.push(newProject.project.id);

    await authedPage.reload();

    const projects = workspaceDetails.workspaceProjects;
    const projectCard = projects.cardByName(newProject.project.name);
    const moveMenuItem = projectCard.actionMenu.move;

    await projectCard.actionMenu.open();
    await moveMenuItem.pwLocator.click();
    await projects.moveModal.destinationWorkspace.pwLocator.fill(destinationWorkspace.name);
    await projects.moveModal.destinationWorkspace.pwLocator.press('Enter');
    await projects.moveModal.footer.submit.pwLocator.click();

    await projects.moveModal.pwLocator.waitFor({ state: 'hidden' });
    await projectCard.pwLocator.waitFor({ state: 'hidden' });

    await workspaceDetails.gotoWorkspace(destinationWorkspace.id);

    await projectCard.pwLocator.waitFor();

    await backgroundApiWorkspace.deleteWorkspace(destinationWorkspace.id);
  });
});
