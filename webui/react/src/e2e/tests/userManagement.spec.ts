import { expect, test } from 'e2e/fixtures/global-fixtures';
import { UserManagement } from 'e2e/models/pages/Admin/UserManagement';
import { SignIn } from 'e2e/models/pages/SignIn';
import { sessionRandomHash } from 'e2e/utils/naming';
import { repeatWithFallback } from 'e2e/utils/polling';
import { saveTestUser } from 'e2e/utils/users';
import { V1PostUserRequest } from 'services/api-ts-sdk/api';

test.describe('User Management', () => {
  // One list of users per test session. This is to encourage a final teardown
  // call of the user fixture to deactivate all users created by each test.
  // Note: This can't collide when running tests in parallel because playwright
  // workers can't share variables.
  const testUsers = new Map<number, V1PostUserRequest>();
  test.beforeEach(async ({ authedPage }) => {
    const userManagementPage = new UserManagement(authedPage);
    await userManagementPage.goto();
    const page1 = userManagementPage.table.table.pagination.pageButtonLocator(1);
    // rows don't load as fast as the rest of the page, so timeout of 10s
    await expect(userManagementPage.table.table.rows.pwLocator).not.toHaveCount(0, {
      timeout: 10_000,
    });
    if (await page1.isVisible()) {
      await expect(
        repeatWithFallback(
          async () => await expect(page1).toHaveClass(/ant-pagination-item-active/),
          async () => await page1.click(),
        ),
      ).toPass({ timeout: 10_000 });
    }
  });

  test.describe('With User Teardown', () => {
    test.afterAll(async ({ backgroundApiUser }) => {
      await test.step('Deactivate Users', async () => {
        for (const [id] of testUsers) {
          await backgroundApiUser.patchUser(id, { active: false });
        }
      });
    });

    test.describe('With a User for Each Test', () => {
      let testUser: V1PostUserRequest;

      test.beforeEach(async ({ user }) => {
        await test.step('Create User', async () => {
          testUser = await user.createUser();
          saveTestUser(testUser, testUsers);
        });
      });

      test('User Table Read', async ({ user }) => {
        await user.validateUser(testUser);
      });

      test('New User Access', async ({ page, auth }) => {
        const userManagementPage = new UserManagement(page);
        await auth.logout();
        await auth.login({ password: testUser.password, username: testUser.user?.username });
        await userManagementPage.nav.sidebar.headerDropdown.open();
        await userManagementPage.nav.sidebar.headerDropdown.settings.pwLocator.waitFor();
        await userManagementPage.nav.sidebar.headerDropdown.admin.pwLocator.waitFor({
          state: 'hidden',
        });
      });

      test('Edit User', async ({ user }) => {
        await test.step('Edit Once', async () => {
          if (testUser.user === undefined) {
            throw new Error('Trying to edit an undefined user.');
          }
          testUser = await user.editUser(testUser, {
            displayName: testUser.user.username + '_edited',
          });
          await user.validateUser(testUser);
        });
        await test.step('Edit Again', async () => {
          testUser = await user.editUser(testUser, { admin: true, displayName: '' });
          await user.validateUser(testUser);
        });
      });
      // DET-10439
      test.skip('Deactivate and Reactivate', async ({ page, user, auth, newAdmin }) => {
        const userManagementPage = new UserManagement(page);
        const signInPage = new SignIn(page);
        await test.step('Deactivate', async () => {
          testUser = await user.changeStatusUser(testUser, false);
          saveTestUser(testUser, testUsers);
          await user.validateUser(testUser);
        });
        await test.step('Attempt Sign In With Deactivated User', async () => {
          await auth.logout();
          await auth.login({
            expectedURL: /login/,
            password: testUser.password,
            username: testUser.user?.username,
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
          await auth.login({
            expectedURL: userManagementPage.url,
            password: newAdmin.password,
            username: newAdmin.user?.username,
          });
          testUser = await user.changeStatusUser(testUser, true);
          saveTestUser(testUser, testUsers);
        });
        await test.step('Successful Sign In', async () => {
          await auth.logout();
          await auth.login({ password: testUser.password, username: testUser.user?.username });
        });
      });
    });

    test.describe('With 10 Users', () => {
      const usernamePrefix = 'test-user-pagination';
      test.beforeAll(async ({ backgroundApiUser }) => {
        await test.step('Create User', async () => {
          // pagination will be 10 per page, so create 11 users
          for (let i = 0; i < 11; i++) {
            const user = await backgroundApiUser.createUser(
              backgroundApiUser.new({ usernamePrefix }),
            );
            saveTestUser(user, testUsers);
          }
        });
      });

      test('[ET-233] Bulk Actions', async ({ page, user, playwright }) => {
        const userManagementPage = new UserManagement(page);

        await test.step('Setup Table Filters', async () => {
          // set pagination to 10
          await expect(
            repeatWithFallback(
              async () => {
                await userManagementPage.table.table.pagination.perPage.openMenu();
                await userManagementPage.table.table.pagination.perPage.perPage10.pwLocator.click();
              },
              async () => {
                // BUG [ET-233]
                await userManagementPage.goto();
              },
            ),
          ).toPass({ timeout: 15_000 });
          // filter by active users
          await userManagementPage.filterStatus.openMenu();
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
        await test.step("Deactivate All Users on the Table's Page (1 User)", async () => {
          await userManagementPage.actions.pwLocator.waitFor({ state: 'hidden' });
          await user.deactivateTestUsersOnTable(testUsers);
        });
        await test.step('Check That the 1 User is Disabled', async () => {
          // wait for table to be stable and check that pagination and "no data" both dont show
          await userManagementPage.table.table.pwLocator.click({ trial: true });
          try {
            await userManagementPage.table.table.noData.pwLocator.waitFor();
            await userManagementPage.table.table.pagination.pwLocator.waitFor();
            // if we see these elements, we should fail the test
            // sometimes BUG [ET-240] makes this test pass unexpectedly
            test.fail();
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

      test('Users Table Row Count matches Users Tab Value', async ({ page }) => {
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
          await test.step(`Compare Table Rows With Pagination ${name}`, async () => {
            await expect(
              repeatWithFallback(
                async () => {
                  await pagination.perPage.openMenu();
                  await paginationOption.pwLocator.click();
                },
                async () => {
                  // BUG [ET-233]
                  await userManagementPage.goto();
                },
              ),
            ).toPass({ timeout: 25_000 });
            await expect(userManagementPage.skeletonTable.pwLocator).not.toBeVisible();
            const paginationSelection = Number(name);
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
