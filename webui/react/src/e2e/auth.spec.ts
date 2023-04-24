import { expect } from '@playwright/test';

import { test } from './global-fixtures';

test.describe('Authentication', () => {
  test.beforeEach(async ({ dev }) => {
    await dev.setServerAddress();
  });

  test('Login and Logout', async ({ page, auth }) => {
    await page.goto('/');

    await test.step('Login steps', async () => {
      await auth.login(/dashboard/);
      await expect(page).toHaveTitle('Home - Determined');
      await expect(page).toHaveURL(/dashboard/);
    });

    await test.step('Logout steps', async () => {
      await auth.logout();
      await expect(page).toHaveTitle('Sign In - Determined');
      await expect(page).toHaveURL(/login/);
    });
  });

  test('Redirect to the target URL after login', async ({ page, auth }) => {
    await page.goto('./models');

    await test.step('Login steps', async () => {
      await auth.login(/models/);
    });

    await expect(page).toHaveTitle('Model Registry - Determined');
    await expect(page).toHaveURL(/models/);
  });
});
