import { expect, type Page } from '@playwright/test';

import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { test } from 'e2e/fixtures/global-fixtures';
import { User, UserFixture } from 'e2e/fixtures/user.fixture';
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
    let expetedRowCount: number;
    await test.step('Get number of users from the tab at the top', async () => {
      const match = (await userManagementPage.userTab.pwLocator.innerText()).match(
        /Users \((\d+)\)/,
      );
      if (match === null) {
        throw new Error('Number not present in tab.');
      }
      expetedRowCount = Number(match[1]);
    });
    for await (const [index, paginationOption] of [
      pagination.perPage.perPage10,
      pagination.perPage.perPage20,
      pagination.perPage.perPage50,
      pagination.perPage.perPage100,
    ].entries()) {
      await test.step(`Compare table rows with pagination:${index}`, async () => {
        await pagination.perPage.pwLocator.click();
        // TODO [INFENG-628] Users page loads slow
        await paginationOption.pwLocator.click({ noWaitAfter: true });
        await paginationOption.pwLocator.waitFor({ state: 'hidden', timeout: 20000 });
        // TODO [INFENG-628] Users page loads slow
        // await paginationOption.pwLocator.click();
        await expect(userManagementPage.skeletonTable.pwLocator).not.toBeVisible();
        const matches = (await pagination.perPage.pwLocator.innerText()).match(/(\d+) \/ page/);
        if (matches === null) {
          throw new Error("Couldn't find pagination selection.");
        }
        const paginationSelection = +matches[1];
        await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(
          Math.min(paginationSelection, expetedRowCount),
        );
      });
    }
  });

  test.describe('With a new User', () => {
    let page: Page;
    let authFixture: AuthFixture;
    let userFixture: UserFixture;
    let userManagementPage: UserManagement;
    let testUser: User;

    test.beforeAll(async ({ browser }) => {
      await test.step('Login', async () => {
        page = await browser.newPage({ recordVideo: { dir: './src/e2e/test-results' } });
        authFixture = new AuthFixture(page);
        userFixture = new UserFixture(page);
        userManagementPage = new UserManagement(page);
        await authFixture.login();
      });

      await test.step('Create User', async () => {
        await userManagementPage.goto();
        testUser = await userFixture.createUser();
      });
    });

    test.afterAll(async () => {
      await userManagementPage.goto();
      await test.step('Deactivate User', async () => {
        await userFixture.deactivateTestUsers();
      });
      await page.close();
    });

    test('User table shows correct data', async () => {
      const { username, id } = testUser;
      await userManagementPage.goto();
      await userManagementPage.search.pwLocator.fill(username);
      await expect(userManagementPage.getRowByID(id).user.pwLocator).toContainText(username);
    });

    test('Edit user', async () => {
      let modifiedUser: User;
      await userManagementPage.goto();
      await test.step('Edit once', async () => {
        modifiedUser = await userFixture.editUser(testUser, {
          displayName: testUser.username + 'mama luigi',
        });
        await userManagementPage.toast.close.pwLocator.click();
        await expect(userManagementPage.toast.pwLocator).toHaveCount(0);
      });
      await test.step('Edit again', async () => {
        testUser = await userFixture.editUser(modifiedUser, { displayName: '', isAdmin: true });
      });
    });
  });
});
