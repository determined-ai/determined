import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { BasePage } from 'e2e/models/BasePage';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { SignIn } from 'e2e/models/pages/SignIn';
import { Workspaces } from 'e2e/models/pages/Workspaces';

test.describe('Navigation', () => {
  const USERNAME = process.env.PW_USER_NAME ?? '';

  test.beforeEach(async ({ dev }) => {
    await dev.setServerAddress();
  });

  test('Top Level', async ({ page, auth }) => {
    await page.goto('/');

    await test.step('Login steps', async () => {
      await auth.login(/dashboard/);
      await expect(page).toHaveTitle(BasePage.getTitle('Home'));
      await expect(page).toHaveURL(/dashboard/);
    });

    await test.step('Navigate to Uncategorized', async () => {
      await page.getByRole('link', { exact: true, name: 'Uncategorized' }).click();
      const expectedURL = /projects\/1\/experiments/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle(BasePage.getTitle('Uncategorized Experiments'));
    });

    await test.step('Navigate to Model Registry', async () => {
      await page.getByRole('link', { name: 'Model Registry' }).click();
      await page.waitForURL(/models/);
      await expect.soft(page).toHaveTitle(BasePage.getTitle('Model Registry'));
    });

    await test.step('Navigate to Tasks', async () => {
      await page.getByRole('link', { name: 'Tasks' }).click();
      const expectedURL = /tasks/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle(BasePage.getTitle('Tasks'));
    });

    await test.step('Navigate to Webhooks', async () => {
      await page.getByRole('link', { name: 'Webhooks' }).click();
      const expectedURL = /webhooks/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle(BasePage.getTitle('Webhooks'));
    });

    await test.step('Navigate to Cluster', async () => {
      await page.getByRole('link', { name: 'Cluster' }).click();
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
      await page.getByRole('navigation').getByText(USERNAME).click();
      await page.getByRole('link', { name: 'Admin' }).click();
      const userManagementPage = new UserManagement(page);
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
