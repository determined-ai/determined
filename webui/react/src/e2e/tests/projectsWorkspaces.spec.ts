import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { Workspaces } from 'e2e/models/pages/Workspaces';

test.describe('Projects', () => {

  test.beforeEach(async ({ dev, auth, page }) => {
    await dev.setServerAddress();
    await auth.login(/dashboard/);
    await expect(page).toHaveTitle('Home - Determined');
    await expect(page).toHaveURL(/dashboard/)
  });

  test('Projects and Workspaces CRUD', async ({ page, auth }) => {
    const workspacesPage = new Workspaces(page);

    await test.step('Navigate to Workspaces', async () => {
      await workspacesPage.nav.sidebar.workspaces.pwLocator.click()
      await page.waitForURL(workspacesPage.url);
      await expect.soft(page).toHaveTitle(workspacesPage.title);
      await expect.soft(page).toHaveURL(workspacesPage.url);
    });

    await test.step('Create a workspace', async () => {
      await workspacesPage.list.newWorkspaceButton.pwLocator.click()
    })
    await test.step('Create a workspace through the sidebar', async () => {})
    await test.step('Create projects', async () => {})
    await test.step('Archive a project', async () => {})
    await test.step('Unarchive a project', async () => {})
    await test.step('Navigation on projects page - sorting and list', async () => {})
    await test.step('Create a model with all possible metadata', async () => {})
    await test.step('Archive a model', async () => {})
    await test.step('Unarchive a model', async () => {})
    await test.step('Move a model between projects', async () => {})
    await test.step('Launch JupyterLab, kill the task, view logs', async () => {})
    await test.step('Navigate with the breadcrum and workspace page', async () => {})
    await test.step('Navigation on workspace page', async () => {})
    await test.step('Navigation to wokspace on the sidebar', async () => {})
    await test.step('Edit a workspace through workspaces page', async () => {})
    await test.step('Edit a workspace through the sidebar', async () => {})
    await test.step('Archive a workspace', async () => {})
    await test.step('Unarchive a workspace', async () => {})
    await test.step('Unpin a workspace through the sidebar', async () => {})
    await test.step('Pin a workspace through the sidebar', async () => {})
    await test.step('Delete a model', async () => {})
    await test.step('Delete a project', async () => {})
    await test.step('Delete a workspace', async () => {})
    await test.step('Delete a workspace through the sidebar', async () => {})

    // await test.step('Navigate to Logout', async () => {
    //   await auth.logout();
    //   await expect.soft(page).toHaveTitle('Sign In - Determined');
    //   await expect.soft(page).toHaveURL(/login/);
    // });
  });
});
