import { expect, type Page } from '@playwright/test';

import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { test } from 'e2e/fixtures/global-fixtures';
import { User, UserFixture } from 'e2e/fixtures/user.fixture';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { SignIn } from 'e2e/models/pages/SignIn';
import { sessionRandomHash } from 'e2e/utils/naming';

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

  test.describe('With New User Teardown', () => {
    let pageSetupTeardown: Page;
    let authFixtureSetupTeardown: AuthFixture;
    let userFixtureSetupTeardown: UserFixture;
    let userManagementPageSetupTeardown: UserManagement;

    test.beforeAll(async ({ browser }) => {
      await test.step('Login', async () => {
        pageSetupTeardown = await browser.newPage();
        authFixtureSetupTeardown = new AuthFixture(pageSetupTeardown);
        userFixtureSetupTeardown = new UserFixture(pageSetupTeardown);
        userManagementPageSetupTeardown = new UserManagement(pageSetupTeardown);
        await authFixtureSetupTeardown.login();
      });
    });

    test.afterAll(async () => {
      await userManagementPageSetupTeardown.goto();
      await test.step('Deactivate User', async () => {
        await userFixtureSetupTeardown.deactivateAllTestUsers();
      });
      await pageSetupTeardown.close();
    });

    test.describe('With a Test User', () => {
      let testUser: User;
      test.beforeAll(async () => {
        await test.step('Create User', async () => {
          await userManagementPageSetupTeardown.goto();
          testUser = await userFixtureSetupTeardown.createUser();
        });
      });

      test('User table shows correct data', async ({ page, user }) => {
        const userManagementPage = new UserManagement(page);
        await userManagementPage.goto();
        await user.validateUser(testUser);
      });

      test('Sign in', async ({ page, auth }) => {
        const userManagementPage = new UserManagement(page);
        await page.goto('/');
        await auth.logout();
        await auth.login({ username: testUser.username });
        await userManagementPage.nav.sidebar.headerDropdown.pwLocator.click();
        await userManagementPage.nav.sidebar.headerDropdown.settings.pwLocator.waitFor();
        await userManagementPage.nav.sidebar.headerDropdown.admin.pwLocator.waitFor({
          state: 'hidden',
        });
      });

      test('Edit user', async ({ page, user }) => {
        const userManagementPage = new UserManagement(page);
        await userManagementPage.goto();
        await test.step('Edit once', async () => {
          testUser = await user.editUser(testUser, {
            displayName: testUser.username + '_edited',
          });
          await user.validateUser(testUser);
        });
        await test.step('Edit again', async () => {
          testUser = await user.editUser(testUser, { displayName: '', isAdmin: true });
          await user.validateUser(testUser);
        });
      });
    });

    test.describe('With Test User we Deactivate', () => {
      let testUser: User;
      test.beforeAll(async () => {
        await test.step('Create User', async () => {
          await userManagementPageSetupTeardown.goto();
          testUser = await userFixtureSetupTeardown.createUser();
        });
      });

      test('Deactivate and Reactivate', async ({ page, user, auth }) => {
        const userManagementPage = new UserManagement(page);
        const signInPage = new SignIn(page);
        await userManagementPage.goto();
        await test.step('Deactivate', async () => {
          testUser = await user.changeStatusUser(testUser, false);
          await user.validateUser(testUser);
        });
        await test.step('Attempt Sign In', async () => {
          await auth.logout();
          await auth.login({ username: testUser.username, waitForURL: /login/ });
          expect(await signInPage.detAuth.errors.message.pwLocator.textContent()).toContain(
            'Login failed',
          );
          expect(await signInPage.detAuth.errors.description.pwLocator.textContent()).toContain(
            'user is not active',
          );
        });
        await test.step('Reactivate', async () => {
          await userManagementPage.goto();
          await auth.login({ waitForURL: userManagementPage.url });
          testUser = await user.changeStatusUser(testUser, true);
        });
        await test.step('Successful Sign In', async () => {
          await auth.logout();
          await auth.login({ username: testUser.username });
        });
      });
    });

    test.describe('With 10 Users', () => {
      const usernamePrefix = 'test-user-pagination';
      test.beforeAll(async () => {
        test.setTimeout(120_000);
        await test.step('Create User', async () => {
          await userManagementPageSetupTeardown.goto();
          await test.step('Create some users', async () => {
            // pagination will be 10 per page, so create 11 users
            for (let i = 0; i < 11; i++) {
              await userFixtureSetupTeardown.createUser({ username: `${usernamePrefix}` });
            }
          });
        });
      });

      test('Group actions, pagination, and filter', async ({ page, user }, testInfo) => {
        const userManagementPage = new UserManagement(page);
        await userManagementPage.goto();

        await test.step('Setup table filters', async () => {
          // set pagination to 10
          await userManagementPage.table.table.pagination.perPage.pwLocator.click();
          await userManagementPage.table.table.pagination.perPage.perPage10.pwLocator.click();
          // filter by active users
          await userManagementPage.filterStatus.pwLocator.click();
          await userManagementPage.filterStatus.activeUsers.pwLocator.click();
          // search for users created this session and wait for table stable
          await userManagementPage.search.pwLocator.fill(usernamePrefix + sessionRandomHash);
          await expect(async () => {
            expect(
              await userManagementPage.table.table.filterRows(async (row) => {
                return (await row.user.name.pwLocator.textContent())?.indexOf(usernamePrefix) === 0;
              }),
            ).toHaveLength(10);
          }).toPass({ timeout: 10000 });
          // go to page 2 to see users
          await userManagementPage.table.table.pagination.pageButtonLocator(2).click();
          await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(1);
        });
        await test.step("Disable all users on the table's page", async () => {
          await user.deactivateTestUsersOnTable();
        });
        // expect this test step to fail
        await test.step('Check that all users are disabled', async () => {
          // wait for table to be stable and check that pagination and "no data" both dont show
          await userManagementPage.table.table.pwLocator.click({ trial: true });
          testInfo.fail(); // BUG [ET-178]
          await userManagementPage.table.table.pagination.pwLocator.waitFor({ state: 'hidden' });
          await userManagementPage.table.table.noData.pwLocator.waitFor({ state: 'hidden' });
          // Expect to see rows from page 1
          await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(10);
        });
      });

      test('Users table count matches admin page users tab', async ({ page }) => {
        test.setTimeout(120_000);
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
        for await (const { name, paginationOption } of [
          {
            name: '10',
            paginationOption: pagination.perPage.perPage10,
          },
          {
            name: '20',
            paginationOption: pagination.perPage.perPage20,
          },
          {
            name: '50',
            paginationOption: pagination.perPage.perPage50,
          },
          {
            name: '100',
            paginationOption: pagination.perPage.perPage100,
          },
        ]) {
          await test.step(`Compare table rows with pagination: ${name}`, async () => {
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
          });
        }
      });
    });
  });
});
