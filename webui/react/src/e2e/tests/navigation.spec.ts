import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { BasePage } from 'e2e/models/BasePage';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { SignIn } from 'e2e/models/pages/SignIn';
import { Workspaces } from 'e2e/models/pages/Workspaces';

test.describe('Navigation', () => {
  test.beforeEach(async ({ dev }) => {
    await dev.setServerAddress();
  });

  test('Sidebar Navigation', async ({ page, auth }) => {
    // we need any page to access the sidebar, and i haven't modeled the homepage yet
    const userManagementPage = new UserManagement(page);

    await page.goto('/');

    await test.step('Login steps', async () => {
      await auth.login();
      await expect(page).toHaveTitle(BasePage.getTitle('Home'));
      await expect(page).toHaveURL(/dashboard/);
    });

    await test.step('Navigate to Uncategorized', async () => {
      await userManagementPage.nav.sidebar.uncategorized.pwLocator.click();
      const expectedURL = /projects\/1\/experiments/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle(BasePage.getTitle('Uncategorized Experiments'));
    });

    await test.step('Navigate to Model Registry', async () => {
      await userManagementPage.nav.sidebar.modelRegistry.pwLocator.click();
      await page.waitForURL(/models/);
      await expect.soft(page).toHaveTitle(BasePage.getTitle('Model Registry'));
    });

    await test.step('Navigate to Tasks', async () => {
      await userManagementPage.nav.sidebar.tasks.pwLocator.click();
      const expectedURL = /tasks/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle(BasePage.getTitle('Tasks'));
    });

    await test.step('Navigate to Webhooks', async () => {
      await userManagementPage.nav.sidebar.webhooks.pwLocator.click();
      const expectedURL = /webhooks/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle(BasePage.getTitle('Webhooks'));
    });

    await test.step('Navigate to Cluster', async () => {
      await userManagementPage.nav.sidebar.cluster.pwLocator.click();
      const expectedURL = /clusters/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle(BasePage.getTitle('Cluster'));
    });

    await test.step('Navigate to Workspaces', async () => {
      const workspacesPage = new Workspaces(page);
      await workspacesPage.nav.sidebar.workspaces.pwLocator.click();
      await page.waitForURL(workspacesPage.getUrlRegExp());
      await expect.soft(page).toHaveTitle(workspacesPage.title);
    });

    await test.step('Navigate to Admin', async () => {
      const userManagementPage = new UserManagement(page);
      await userManagementPage.nav.sidebar.headerDropdown.pwLocator.click();
      await userManagementPage.nav.sidebar.headerDropdown.admin.pwLocator.click();
      await page.waitForURL(userManagementPage.getUrlRegExp());
      await expect.soft(page).toHaveTitle(userManagementPage.title);
    });

    await test.step('Navigate to Logout', async () => {
      await auth.logout();
      const signInPage = new SignIn(page);
      await expect.soft(page).toHaveTitle(signInPage.title);
      await expect.soft(page).toHaveURL(signInPage.getUrlRegExp());
    });
  });
});
