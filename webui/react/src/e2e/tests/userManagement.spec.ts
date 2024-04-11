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

  test('Users table count matches admin page users tab', async ({ page }) => {
    const userManagementPage = new UserManagement(page);
    await userManagementPage.goto();
    const pagination = userManagementPage.table.table.pagination;
    const match = (await userManagementPage.userTab.pwLocator.innerText()).match(/\d+/);
    if (match === null) {
      throw new Error('Number not present in tab.');
    }
    const expetedRowCount = +match[0];
    for (const paginationOption of [
      pagination.perPage.perPage10,
      pagination.perPage.perPage20,
      pagination.perPage.perPage50,
      pagination.perPage.perPage100,
    ]) {
      await pagination.perPage.pwLocator.click();
      await paginationOption.pwLocator.click();
      await expect(userManagementPage.skeletonTable.pwLocator).not.toBeVisible();
      const matches = (await pagination.perPage.pwLocator.innerText()).match(/(\d+) \/ page/);
      if (matches === null) {
        throw new Error("Couldn't find pagination selection.");
      }
      const paginationSelection = +matches[1];
      await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(
        Math.min(paginationSelection, expetedRowCount),
      );
    }
  });

  test.describe('With a new User', () => {
    let page: Page;
    let authFixture: AuthFixture;
    let userManagementPage: UserManagement;
    let userid: string;
    let username: string;

    test.beforeAll(async ({ browser }) => {
      username = 'test-user-' + uuidv4();

      await test.step('Login', async () => {
        page = await browser.newPage();
        authFixture = new AuthFixture(page);
        userManagementPage = new UserManagement(page);
        await authFixture.login();
      });

      await test.step('Create a user', async () => {
        await userManagementPage.goto();
        await userManagementPage.addUser.pwLocator.click();
        await expect(userManagementPage.createUserModal.pwLocator).toBeVisible();
        await userManagementPage.createUserModal.username.pwLocator.fill(username);
        await userManagementPage.createUserModal.footer.submit.pwLocator.click();
      });

      await test.step('Set the user id', async () => {
        await userManagementPage.search.pwLocator.fill(username);
        await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(1);
        userid = await (await userManagementPage.getRowByUsername(username)).getID();
      });
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
