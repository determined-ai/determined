
import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';

test.describe('Authentication', () => {
  test.beforeEach(async ({ auth, dev }) => {
    await dev.setServerAddress();
    await auth.login()
  });

  test('Navigate to User Management', async ({ page, dev }) => {
    const userManagementPage = new UserManagement(page);
    await userManagementPage.nav.sidebar.headerDropdown.pwLocator.click()
    await userManagementPage.nav.sidebar.headerDropdownMenuItemsAdmin.pwLocator.click()
  });

  // test('Redirect to the target URL after login', async ({ page, auth }) => {
  //   await test.step('Visit a page and expect redirect back to login', async () => {
  //     await page.goto('./models');
  //     await expect(page).toHaveURL(/login/);
  //   });

  //   await test.step('Login and expect redirect to previous page', async () => {
  //     await auth.login(/models/);
  //     await expect(page).toHaveTitle('Model Registry - Determined');
  //   });
  // });
});
  