import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { BasePage } from 'e2e/models/BasePage';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { SignIn } from 'e2e/models/pages/SignIn';
import { Workspaces } from 'e2e/models/pages/Workspaces';

test.describe('Navigation', () => {
  test('Sidebar Navigation', async ({ authedPage, auth }) => {
    // we need any page to access the sidebar, and i haven't modeled the homepage yet
    const userManagementPage = new UserManagement(authedPage);

    await test.step('Login steps', async () => {
      await expect(authedPage).toHaveTitle(BasePage.getTitle('Home'));
      await expect(authedPage).toHaveURL(/dashboard/);
    });

    await test.step('Navigate to Uncategorized', async () => {
      await userManagementPage.nav.sidebar.uncategorized.pwLocator.click();
      const expectedURL = /projects\/1\/experiments/;
      await authedPage.waitForURL(expectedURL);
      await expect.soft(authedPage).toHaveTitle(BasePage.getTitle('Uncategorized Experiments'));
    });

    await test.step('Navigate to Model Registry', async () => {
      await userManagementPage.nav.sidebar.modelRegistry.pwLocator.click();
      await authedPage.waitForURL(/models/);
      await expect.soft(authedPage).toHaveTitle(BasePage.getTitle('Model Registry'));
    });

    await test.step('Navigate to Tasks', async () => {
      await userManagementPage.nav.sidebar.tasks.pwLocator.click();
      const expectedURL = /tasks/;
      await authedPage.waitForURL(expectedURL);
      await expect.soft(authedPage).toHaveTitle(BasePage.getTitle('Tasks'));
    });

    await test.step('Navigate to Webhooks', async () => {
      await userManagementPage.nav.sidebar.webhooks.pwLocator.click();
      const expectedURL = /webhooks/;
      await authedPage.waitForURL(expectedURL);
      await expect.soft(authedPage).toHaveTitle(BasePage.getTitle('Webhooks'));
    });

    await test.step('Navigate to Cluster', async () => {
      await userManagementPage.nav.sidebar.cluster.pwLocator.click();
      const expectedURL = /clusters/;
      await authedPage.waitForURL(expectedURL);
      await expect.soft(authedPage).toHaveTitle(BasePage.getTitle('Cluster'));
    });

    await test.step('Navigate to Workspaces', async () => {
      const workspacesPage = new Workspaces(authedPage);
      await workspacesPage.nav.sidebar.workspaces.pwLocator.click();
      await authedPage.waitForURL(workspacesPage.getUrlRegExp());
      await expect.soft(authedPage).toHaveTitle(workspacesPage.title);
    });

    await test.step('Navigate to Admin', async () => {
      const userManagementPage = new UserManagement(authedPage);
      await userManagementPage.nav.sidebar.headerDropdown.open();
      await userManagementPage.nav.sidebar.headerDropdown.admin.pwLocator.click();
      await authedPage.waitForURL(userManagementPage.getUrlRegExp());
      await expect.soft(authedPage).toHaveTitle(userManagementPage.title);
    });

    await test.step('Navigate to Logout', async () => {
      await auth.logout();
      const signInPage = new SignIn(authedPage);
      await expect.soft(authedPage).toHaveTitle(signInPage.title);
      await expect.soft(authedPage).toHaveURL(signInPage.getUrlRegExp());
    });
  });
});
