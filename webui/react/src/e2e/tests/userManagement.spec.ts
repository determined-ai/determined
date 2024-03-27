
import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';

test.describe('User Management', () => {
  test.beforeEach(async ({ auth, dev }) => {
    await dev.setServerAddress();
    await auth.login()
  });

  test('Navigate to User Management', async ({ page }) => {
    const userManagementPage = new UserManagement(page);
    await userManagementPage.nav.sidebar.headerDropdown.pwLocator.click()
    await userManagementPage.nav.sidebar.headerDropdown.admin.pwLocator.click()
    await expect(page).toHaveTitle(UserManagement.title);
    await expect(page).toHaveURL(UserManagement.url);
  });
});
  