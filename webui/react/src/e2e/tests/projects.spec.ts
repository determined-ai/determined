import _ from 'lodash';

import { expect, test } from 'e2e/fixtures/global-fixtures';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';
import { WorkspaceDetails } from 'e2e/models/pages/WorkspaceDetails';
import { WorkspaceProjects } from 'e2e/models/pages/WorkspaceDetails/WorkspaceProjects';
import { randId, safeName } from 'e2e/utils/naming';
import { V1Project } from 'services/api-ts-sdk';

const getCurrentProjectNames = async (workspaceProjects: WorkspaceProjects) => {
  await workspaceProjects.projectCards.pwLocator.nth(0).waitFor();

  const cardTitles = await workspaceProjects.projectCards.title.pwLocator.all();
  return await Promise.all(
    cardTitles.map(async (title) => {
      return await title.textContent();
    }),
  );
};

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

    const workspaceProjects = workspaceDetails.workspaceProjects;

    await test.step('Create a Project', async () => {
      await workspaceProjects.newProject.pwLocator.click();
      await workspaceProjects.createModal.projectName.pwLocator.fill(projectName);
      await workspaceProjects.createModal.description.pwLocator.fill(randId());
      await workspaceProjects.createModal.footer.submit.pwLocator.click();
      projectIds.push(await projectDetails.getIdFromUrl());
      await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
      await workspaceProjects.cardByName(projectName).pwLocator.waitFor();
    });

    await test.step('Delete a Project', async () => {
      await workspaceDetails.gotoWorkspace(newWorkspace.response.workspace.id);
      await workspaceDetails.projectsTab.pwLocator.click();
      const projectCard = workspaceProjects.cardByName(projectName);
      await projectCard.actionMenu.open();
      await projectCard.actionMenu.delete.pwLocator.click();
      await workspaceProjects.deleteModal.nameConfirmation.pwLocator.fill(projectName);
      await workspaceProjects.deleteModal.footer.submit.pwLocator.click();
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

    const workspaceProjects = workspaceDetails.workspaceProjects;
    const projectCard = workspaceProjects.cardByName(newProject.project.name);
    const moveMenuItem = projectCard.actionMenu.move;

    await projectCard.actionMenu.open();
    await moveMenuItem.pwLocator.click();
    await workspaceProjects.moveModal.destinationWorkspace.pwLocator.fill(
      destinationWorkspace.name,
    );
    await workspaceProjects.moveModal.destinationWorkspace.pwLocator.press('Enter');
    await workspaceProjects.moveModal.footer.submit.pwLocator.click();

    await workspaceProjects.moveModal.pwLocator.waitFor({ state: 'hidden' });
    await projectCard.pwLocator.waitFor({ state: 'hidden' });

    await workspaceDetails.gotoWorkspace(destinationWorkspace.id);

    await projectCard.pwLocator.waitFor();

    await backgroundApiWorkspace.deleteWorkspace(destinationWorkspace.id);
  });
});

test.describe('Project List', () => {
  const projects: V1Project[] = [];
  const workspaceIds: number[] = [];

  test.beforeAll(async ({ backgroundApiProject, backgroundApiWorkspace }) => {
    // Shared 'newWorkspace' fixture can sometimes contain projects from other test runs.
    // Avoid this issue by creating a workspace specifically for Project List tests:
    workspaceIds.push(
      (await backgroundApiWorkspace.createWorkspace(backgroundApiWorkspace.new())).workspace.id,
    );

    const olderProject = (
      await backgroundApiProject.createProject(workspaceIds[0], {
        name: safeName('a-test-project'),
        workspaceId: workspaceIds[0],
      })
    ).project;
    projects.push(olderProject);

    const newerProject = (
      await backgroundApiProject.createProject(workspaceIds[0], {
        name: safeName('b-test-project'),
        workspaceId: workspaceIds[0],
      })
    ).project;
    projects.push(newerProject);
  });

  test.beforeEach(async ({ authedPage }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);
    await workspaceDetails.gotoWorkspace(workspaceIds[0]);
    const workspaceProjects = workspaceDetails.workspaceProjects;

    await workspaceProjects.whoseSelect.selectMenuOption('All Projects');
    await workspaceProjects.sortSelect.selectMenuOption('Newest to Oldest');
    await workspaceProjects.gridListRadioGroup.grid.pwLocator.click();
  });

  test.afterAll(async ({ backgroundApiProject, backgroundApiWorkspace }) => {
    for (const project of projects) {
      await backgroundApiProject.deleteProject(project.id);
    }
    for (const workspaceId of workspaceIds) {
      await backgroundApiWorkspace.deleteWorkspace(workspaceId);
    }
  });

  test('Sort', async ({ authedPage }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);
    const workspaceProjects = workspaceDetails.workspaceProjects;

    const namesAfterNewest = await getCurrentProjectNames(workspaceProjects);
    const idSortedProjectNames = _.orderBy(projects, 'id', 'desc').map((p) => p.name);
    expect(idSortedProjectNames).toEqual(
      namesAfterNewest.filter((n) => {
        return n && projects.map((p) => p.name).includes(n);
      }),
    );

    await workspaceProjects.sortSelect.selectMenuOption('Alphabetical');

    const namesAfterAlphabetical = await getCurrentProjectNames(workspaceProjects);
    const nameSortedProjectNames = _.orderBy(projects, 'name', 'asc').map((p) => p.name);
    expect(nameSortedProjectNames).toEqual(
      namesAfterAlphabetical.filter((n) => {
        return n && projects.map((p) => p.name).includes(n);
      }),
    );
  });

  test('Filter', async ({ authedPage, apiProject }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);
    const workspaceProjects = workspaceDetails.workspaceProjects;

    const currentUserProject = (
      await apiProject.createProject(workspaceIds[0], {
        name: safeName('current-user-project'),
        workspaceId: workspaceIds[0],
      })
    ).project;

    const currentUserProjectName = currentUserProject.name;
    const otherUserProjectName = projects.map((p) => p.name)[0];

    await authedPage.reload();

    const namesAfterAll = await getCurrentProjectNames(workspaceProjects);
    expect(namesAfterAll).toContain(otherUserProjectName);
    expect(namesAfterAll).toContain(currentUserProjectName);

    await workspaceProjects.whoseSelect.selectMenuOption("Others' Projects");
    const namesAfterOthers = await getCurrentProjectNames(workspaceProjects);
    expect(namesAfterOthers).toContain(otherUserProjectName);
    expect(namesAfterOthers).not.toContain(currentUserProjectName);

    await workspaceProjects.whoseSelect.selectMenuOption('My Projects');
    const namesAfterMy = await getCurrentProjectNames(workspaceProjects);
    expect(namesAfterMy).toContain(currentUserProjectName);
    expect(namesAfterMy).not.toContain(otherUserProjectName);

    await apiProject.deleteProject(currentUserProject.id);
  });

  test('View Toggle', async ({ authedPage }) => {
    const workspaceDetails = new WorkspaceDetails(authedPage);
    const workspaceProjects = workspaceDetails.workspaceProjects;

    const firstCard = workspaceProjects.projectCards.nth(0);
    const firstRow = workspaceProjects.table.table.rows.nth(0);

    await workspaceProjects.gridListRadioGroup.list.pwLocator.click();
    await firstCard.pwLocator.waitFor({ state: 'hidden' });
    await firstRow.pwLocator.waitFor();

    await workspaceProjects.gridListRadioGroup.grid.pwLocator.click();
    await firstRow.pwLocator.waitFor({ state: 'hidden' });
    await firstCard.pwLocator.waitFor();
  });
});
