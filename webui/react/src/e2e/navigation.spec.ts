import { expect } from '@playwright/test';

import { test } from 'e2e/global-fixtures';

test.describe('Navigation', () => {
  const USERNAME = process.env.PW_USER_NAME ?? '';

  test.beforeEach(async ({ dev }) => {
    await dev.setServerAddress();
  });

  test('Top Level', async ({ page, auth }) => {
    await page.goto('/');

    await test.step('Login steps', async () => {
      await auth.login(/dashboard/);
      await expect(page).toHaveTitle('Home - Determined');
      await expect(page).toHaveURL(/dashboard/);
    });

    await test.step('Navigate to Uncategorized', async () => {
      await page.getByRole('link', { exact: true, name: 'Uncategorized' }).click();
      const expectedURL = /projects\/1\/experiments/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle('Uncategorized Experiments - Determined');
      await expect.soft(page).toHaveURL(expectedURL);
    });

    await test.step('Navigate to Model Registry', async () => {
      await page.getByRole('link', { name: 'Model Registry' }).click();
      const expectedURL = /models/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle('Model Registry - Determined');
      await expect.soft(page).toHaveURL(expectedURL);
    });

    await test.step('Navigate to Tasks', async () => {
      await page.getByRole('link', { name: 'Tasks' }).click();
      const expectedURL = /tasks/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle('Tasks - Determined');
      await expect.soft(page).toHaveURL(expectedURL);
    });

    await test.step('Navigate to Webhooks', async () => {
      await page.getByRole('link', { name: 'Webhooks' }).click();
      const expectedURL = /webhooks/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle('Webhooks - Determined');
      await expect.soft(page).toHaveURL(expectedURL);
    });

    await test.step('Navigate to Cluster', async () => {
      await page.getByRole('link', { name: 'Cluster' }).click();
      const expectedURL = /clusters/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle('Cluster - Determined');
      await expect.soft(page).toHaveURL(expectedURL);
    });

    await test.step('Navigate to Workspaces', async () => {
      await page.getByRole('link', { name: 'Workspaces' }).click();
      const expectedURL = /workspaces/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle('Workspaces - Determined');
      await expect.soft(page).toHaveURL(expectedURL);
    });

    await test.step('Navigate to Admin', async () => {
      await page.getByRole('navigation').getByText(USERNAME).click();
      await page.getByRole('link', { name: 'Admin' }).click();
      const expectedURL = /admin\/user-management/;
      await page.waitForURL(expectedURL);
      await expect.soft(page).toHaveTitle('Determined');
      await expect.soft(page).toHaveURL(expectedURL);
    });

    await test.step('Navigate to Logout', async () => {
      await auth.logout();
      await expect.soft(page).toHaveTitle('Sign In - Determined');
      await expect.soft(page).toHaveURL(/login/);
    });
  });
});
