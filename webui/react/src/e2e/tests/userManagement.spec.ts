import { expect } from '@playwright/test';
import playwright from 'playwright';

import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { test } from 'e2e/fixtures/global-fixtures';
import { User, UserFixture } from 'e2e/fixtures/user.fixture';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { SignIn } from 'e2e/models/pages/SignIn';
import { sessionRandomHash } from 'e2e/utils/naming';
import { repeatWithFallback } from 'e2e/utils/polling';

// creating users while running tests in parallel can cause the users table to refresh at unexpected times
test.describe.configure({ mode: 'serial' });

test.describe('User Management', () => {
  test.beforeEach(async ({ auth, dev, page }) => {
    await dev.setServerAddress();
    await auth.login();
    const userManagementPage = new UserManagement(page);
    await userManagementPage.goto();

    // wait for table to be stable and select page 1
    await userManagementPage.table.table.rows.pwLocator.count();
    const page1 = userManagementPage.table.table.pagination.pageButtonLocator(1);
    if (await userManagementPage.table.table.pagination.pwLocator.isVisible()) {
      await expect(
        repeatWithFallback(
          async () => {
            await expect(page1).toHaveClass(/ant-pagination-item-active/);
          },
          async () => {
            await page1.click();
          },
        ),
      ).toPass({ timeout: 10_000 });
    }
  });

  test('Navigate to User Management', async ({ page }) => {
    const userManagementPage = new UserManagement(page);
    await userManagementPage.nav.sidebar.headerDropdown.pwLocator.click();
    await userManagementPage.nav.sidebar.headerDropdown.admin.pwLocator.click();
    await expect(page).toHaveTitle(userManagementPage.title);
    await expect(page).toHaveURL(userManagementPage.url);
  });

  test.describe('With New User Teardown', () => {
    test.afterAll(async ({ browser }) => {
      const pageSetupTeardown = await browser.newPage();
      const authFixtureSetupTeardown = new AuthFixture(pageSetupTeardown);
      const userFixtureSetupTeardown = new UserFixture(pageSetupTeardown);
      const userManagementPageSetupTeardown = new UserManagement(pageSetupTeardown);
      await authFixtureSetupTeardown.login();
      await test.step('Deactivate Users', async () => {
        await userManagementPageSetupTeardown.goto();
        await userFixtureSetupTeardown.deactivateAllTestUsers();
      });
      await pageSetupTeardown.close();
    });

    test.describe('With a Test User', () => {
      let testUser: User;
      test.beforeAll(async ({ browser }) => {
        const pageSetupTeardown = await browser.newPage();
        const authFixtureSetupTeardown = new AuthFixture(pageSetupTeardown);
        const userFixtureSetupTeardown = new UserFixture(pageSetupTeardown);
        const userManagementPageSetupTeardown = new UserManagement(pageSetupTeardown);
        await authFixtureSetupTeardown.login();
        await test.step('Create User', async () => {
          await userManagementPageSetupTeardown.goto();
          testUser = await userFixtureSetupTeardown.createUser();
        });
        await authFixtureSetupTeardown.logout();
        await pageSetupTeardown.close();
      });

      test('User table shows correct data', async ({ user }) => {
        await user.validateUser(testUser);
      });

      test('New user acess', async ({ page, auth }) => {
        const userManagementPage = new UserManagement(page);
        await auth.logout();
        await auth.login(testUser);
        await userManagementPage.nav.sidebar.headerDropdown.pwLocator.click();
        await userManagementPage.nav.sidebar.headerDropdown.settings.pwLocator.waitFor();
        await userManagementPage.nav.sidebar.headerDropdown.admin.pwLocator.waitFor({
          state: 'hidden',
        });
      });

      test('Edit user', async ({ user }) => {
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

      test.beforeAll(async ({ browser }) => {
        const pageSetupTeardown = await browser.newPage();
        const authFixtureSetupTeardown = new AuthFixture(pageSetupTeardown);
        const userFixtureSetupTeardown = new UserFixture(pageSetupTeardown);
        const userManagementPageSetupTeardown = new UserManagement(pageSetupTeardown);
        await authFixtureSetupTeardown.login();
        await test.step('Create User', async () => {
          await userManagementPageSetupTeardown.goto();
          testUser = await userFixtureSetupTeardown.createUser();
        });
        await authFixtureSetupTeardown.logout();
        await pageSetupTeardown.close();
      });

      test('Deactivate and Reactivate', async ({ page, user, auth }) => {
        // test does does three and a half logins, so we need to increase the timeout
        test.setTimeout(120_000);
        const userManagementPage = new UserManagement(page);
        const signInPage = new SignIn(page);
        await test.step('Deactivate', async () => {
          testUser = await user.changeStatusUser(testUser, false);
          await user.validateUser(testUser);
        });
        await test.step('Attempt Sign In', async () => {
          await auth.logout();
          await auth.login({
            password: testUser.password,
            username: testUser.username,
            waitForURL: /login/,
          });
          expect(await signInPage.detAuth.errors.message.pwLocator.textContent()).toContain(
            'Login failed',
          );
          expect(await signInPage.detAuth.errors.description.pwLocator.textContent()).toContain(
            'user is not active',
          );
        });
        await test.step('Reactivate', async () => {
          await userManagementPage.goto({ verify: false });
          // TODO the verify false on the line above isn't working as expected
          // if we don't expect this url, the automation runs too fast and login
          // thinks we've already logged in, skipping the login automation.
          // We might need to find a way to be more explicit about the page state.
          await expect(page).toHaveURL(/login/);
          await auth.login({ waitForURL: userManagementPage.url });
          testUser = await user.changeStatusUser(testUser, true);
        });
        await test.step('Successful Sign In', async () => {
          await auth.logout();
          await auth.login(testUser);
        });
      });
    });

    test.describe('With 10 Users', () => {
      const usernamePrefix = 'test-user-pagination';
      test.beforeAll(async ({ browser }) => {
        test.setTimeout(180_000);
        const pageSetupTeardown = await browser.newPage();
        const authFixtureSetupTeardown = new AuthFixture(pageSetupTeardown);
        const userFixtureSetupTeardown = new UserFixture(pageSetupTeardown);
        const userManagementPageSetupTeardown = new UserManagement(pageSetupTeardown);
        await authFixtureSetupTeardown.login();
        await test.step('Create User', async () => {
          await userManagementPageSetupTeardown.goto();
          // pagination will be 10 per page, so create 11 users
          for (let i = 0; i < 11; i++) {
            await userFixtureSetupTeardown.createUser({ username: `${usernamePrefix}` });
          }
        });
        await authFixtureSetupTeardown.logout();
        await pageSetupTeardown.close();
      });

      test.skip('[ET-233, ET-178] Bulk actions', async ({ page, user }, testInfo) => {
        const userManagementPage = new UserManagement(page);

        await test.step('Setup table filters', async () => {
          // set pagination to 10
          await expect(
            repeatWithFallback(
              async () => {
                await userManagementPage.table.table.pagination.perPage.pwLocator.click();
                await userManagementPage.table.table.pagination.perPage.perPage10.pwLocator.click();
              },
              async () => {
                // BUG [ET-233]
                await userManagementPage.goto();
              },
            ),
          ).toPass({ timeout: 15_000 });
          // filter by active users
          await userManagementPage.filterStatus.pwLocator.click();
          await userManagementPage.filterStatus.activeUsers.pwLocator.click();
          await expect(async () => {
            expect(
              await userManagementPage.table.table.filterRows(async (row) => {
                return (await row.status.pwLocator.textContent()) === 'Active';
              }),
            ).toHaveLength(10);
          }).toPass({ timeout: 10_000 });
          // search for users created this session and wait for table stable
          await userManagementPage.search.pwLocator.fill(usernamePrefix + sessionRandomHash);
          await expect(async () => {
            expect(
              await userManagementPage.table.table.filterRows(async (row) => {
                return (await row.user.name.pwLocator.textContent())?.indexOf(usernamePrefix) === 0;
              }),
            ).toHaveLength(10);
          }).toPass({ timeout: 10_000 });
          // go to page 2 to see users
          await expect(async () => {
            // BUG [ET-240]
            await userManagementPage.table.table.pagination.pageButtonLocator(2).click();
            await expect(
              userManagementPage.table.table.pagination.pageButtonLocator(2),
            ).toHaveClass(/ant-pagination-item-active/);
            await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(1, {
              timeout: 2_000,
            });
          }).toPass({ timeout: 10_000 });
        });
        await test.step("Disable all users on the table's page", async () => {
          await userManagementPage.actions.pwLocator.waitFor({ state: 'hidden' });
          await user.deactivateTestUsersOnTable();
        });
        // expect this test step to fail
        await test.step('Check that all users are disabled', async () => {
          // wait for table to be stable and check that pagination and "no data" both dont show
          await userManagementPage.table.table.pwLocator.click({ trial: true });
          testInfo.fail(); // BUG [ET-178]
          try {
            await userManagementPage.table.table.noData.pwLocator.waitFor();
            await userManagementPage.table.table.pagination.pwLocator.waitFor();
            // if we see these elements, we should fail the test
            // sometimes BUG [ET-240] makes this test pass unexpectedly
            throw new Error('Expected table to have data and no pagination');
          } catch (error) {
            // if we see a timeout error, that means we don't see "no data"
            if (!(error instanceof playwright.errors.TimeoutError)) {
              // if we see any other error, we should still fail the test
              throw error;
            }
          }
          // Expect to see rows from page 1
          await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(10);
        });
      });

      test('Users table count matches admin page users tab', async ({ page }) => {
        test.setTimeout(120_000);
        const userManagementPage = new UserManagement(page);
        const getExpectedRowCount = async (): Promise<number> => {
          const match = (await userManagementPage.userTab.pwLocator.innerText()).match(
            /Users \((\d+)\)/,
          );
          if (match === null) {
            throw new Error('Number not present in tab.');
          }
          return Number(match[1]);
        };

        const pagination = userManagementPage.table.table.pagination;
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
            await expect(
              repeatWithFallback(
                async () => {
                  await pagination.perPage.pwLocator.click();
                  await paginationOption.pwLocator.click();
                },
                async () => {
                  // BUG [ET-233]
                  await userManagementPage.goto();
                },
              ),
            ).toPass({ timeout: 25_000 });
            await expect(userManagementPage.skeletonTable.pwLocator).not.toBeVisible();
            const matches = (await pagination.perPage.pwLocator.innerText()).match(/(\d+) \/ page/);
            if (matches === null) {
              throw new Error("Couldn't find pagination selection.");
            }
            const paginationSelection = Number(matches[1]);
            await expect(
              repeatWithFallback(
                async () => {
                  // grab the count of the table rows and big number at the top at the same time
                  // in case the table refreshes with more users during a parallel run
                  await expect(userManagementPage.table.table.rows.pwLocator).toHaveCount(
                    Math.min(paginationSelection, await getExpectedRowCount()),
                  );
                },
                async () => {
                  // if the above doesn't pass, refresh the page and try again. This is to handle
                  // the case where the table refreshes with more users, but the other number hasn't refreshed yet
                  await userManagementPage.goto();
                },
              ),
            ).toPass({ timeout: 20_000 });
          });
        }
      });
    });
  });
});
