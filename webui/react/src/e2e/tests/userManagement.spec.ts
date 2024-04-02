import { expect, type Page } from '@playwright/test';
import { v4 as uuidv4 } from 'uuid';

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

  // TODO test table order and explain why

  test.describe('With a new User', () => {
    let page: Page;
    let authFixture: AuthFixture;
    let userManagementPage: UserManagement;
    let userid: string;
    const username = 'test-user-' + uuidv4();

    test.beforeAll(async ({ browser }) => {
      page = await browser.newPage();
      authFixture = new AuthFixture(page);
      userManagementPage = new UserManagement(page);
      await authFixture.login();

      await userManagementPage.goto();
      await expect(userManagementPage.userTab.pwLocator).toContainText(/\d+/)
      const expetedRowCount = +(await userManagementPage.userTab.pwLocator.innerText()).match(/\d+/)![0]
      await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(Math.min(10, expetedRowCount));
      // const oldIDs = await userManagementPage.table.table.allRowKeys();
      await userManagementPage.addUser.pwLocator.click();
      await expect(userManagementPage.createUserModal.pwLocator).toBeVisible();
      await userManagementPage.createUserModal.username.pwLocator.fill(username);
      await userManagementPage.createUserModal.footer.submit.pwLocator.click();
      // await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(oldIDs.length + 1);
      // const newIDs = await userManagementPage.table.table.newRowKeys(oldIDs);
      await userManagementPage.search.pwLocator.fill(username);
      userid = await (await userManagementPage.filterRowsByUsername(username)).getID()
    });
    test.afterAll(async () => {
      if (userid !== undefined) {
        await userManagementPage.getRowByID(userid).actions.pwLocator.click();
        await userManagementPage.getRowByID(userid).actions.state.pwLocator.click();
      }
      await page.close();
    });
    test('User table shows correct name', async () => {
      await expect(userManagementPage.getRowByID(userid).user.pwLocator).toContainText(username);
    });
  });
});
