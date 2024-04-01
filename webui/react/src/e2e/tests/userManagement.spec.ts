import { expect, type Page } from '@playwright/test';
import { uuid } from 'fast-check';

import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { test } from 'e2e/fixtures/global-fixtures';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';

test.describe('User Management', () => {
  test.beforeEach(async ({ auth, dev }) => {
    await dev.setServerAddress();
    await auth.login();
  });

  test('Navigate to User Management', async ({ page }) => {
    const userManagementPage = new UserManagement(page);
    await userManagementPage.nav.sidebar.headerDropdown.pwLocator.click();
    await userManagementPage.nav.sidebar.headerDropdown.admin.pwLocator.click();
    await expect(page).toHaveTitle(userManagementPage.title);
    await expect(page).toHaveURL(userManagementPage.url);
  });

  test.describe('With a new User', () => {
    let page: Page;
    let userid: string;
    let authFixture: AuthFixture;
    let userManagementPage: UserManagement;
    const username = 'test-user' + uuid();

    test.beforeAll(async ({ browser }) => {
      page = await browser.newPage();
      authFixture = new AuthFixture(page);
      userManagementPage = new UserManagement(page);
      await authFixture.login();

      await userManagementPage.goto();
      await expect(userManagementPage.table.pwLocator).toBeVisible();
      const oldIDs = await userManagementPage.table.table.allRowKeys();
      await userManagementPage.addUser.pwLocator.click();
      await expect(userManagementPage.createUserModal.pwLocator).toBeVisible();
      await userManagementPage.createUserModal.username.pwLocator.fill(username);
      await userManagementPage.createUserModal.footer.submit.pwLocator.click();
      const newIDs = await userManagementPage.table.table.newRowKeys(oldIDs);
      expect(newIDs, `newids ${newIDs}, oldids ${oldIDs}`).toHaveLength(1);
      userid = newIDs[0];
    });
    test.afterAll(async () => {
      if (userid !== undefined) {
        const userManagementPage = new UserManagement(page);
        await userManagementPage.getRowByID(userid).actions.pwLocator.click();
        await userManagementPage.getRowByID(userid).actions.state.pwLocator.click();
      }
      await page.close();
    });
    test('Navigate to User Management', async () => {
      const userManagementPage = new UserManagement(page);
      await expect(userManagementPage.getRowByID(userid).user.pwLocator).toContainText(username);
    });
  });
});
