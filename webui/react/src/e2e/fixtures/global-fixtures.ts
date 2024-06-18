import { test as base, Page } from '@playwright/test';

import { safeName } from 'e2e/utils/naming';
import { V1PostUserRequest } from 'services/api-ts-sdk/api';

// eslint-disable-next-line no-restricted-imports
import playwrightConfig from '../../../playwright.config';

import { ApiAuthFixture } from './api.auth.fixture';
import { ApiUserFixture } from './api.user.fixture';
import { AuthFixture } from './auth.fixture';
import { DevFixture } from './dev.fixture';
import { UserFixture } from './user.fixture';

type CustomFixtures = {
  dev: DevFixture;
  auth: AuthFixture;
  apiAuth: ApiAuthFixture;
  user: UserFixture;
  apiUser: ApiUserFixture;
  authedPage: Page;
};

type CustomWorkerFixtures = {
  newAdmin: V1PostUserRequest;
  backgroundApiAuth: ApiAuthFixture;
  backgroundApiUser: ApiUserFixture;
};

// https://playwright.dev/docs/test-fixtures
export const test = base.extend<CustomFixtures, CustomWorkerFixtures>({
  // get the auth but allow yourself to log in through the api manually.
  apiAuth: async ({ playwright, browser, dev, baseURL, newAdmin }, use) => {
    await dev.setServerAddress();
    const apiAuth = new ApiAuthFixture(playwright.request, browser, baseURL, dev.page);
    await apiAuth.login({
      creds: {
        password: newAdmin.password!,
        username: newAdmin.user!.username,
      },
    });
    await use(apiAuth);
  },

  apiUser: async ({ apiAuth }, use) => {
    const apiUser = new ApiUserFixture(apiAuth);
    await use(apiUser);
  },

  auth: async ({ page }, use) => {
    const auth = new AuthFixture(page);
    await use(auth);
  },

  // get the existing page but with auth cookie already logged in
  authedPage: async ({ apiAuth }, use) => {
    await use(apiAuth.page);
  },

  /**
   * Does not require the pre-existing Playwright page and does not login so this can be called in beforeAll.
   * Generally use another api fixture instead if you want to call an api. If you just want a logged-in page,
   * use apiAuth in beforeEach().
   */
  backgroundApiAuth: [
    async ({ playwright, browser }, use) => {
      const backgroundApiAuth = new ApiAuthFixture(
        playwright.request,
        browser,
        playwrightConfig.use?.baseURL,
      );
      await backgroundApiAuth.login();
      await use(backgroundApiAuth);
      await backgroundApiAuth.dispose();
    },
    { scope: 'worker' },
  ],
  /**
   * Allows calling the user api without a page so that it can run in beforeAll(). You will need to get a bearer
   * token by calling backgroundApiUser.apiAuth.login(). This will also provision a page in the background which
   * will be disposed of logout(). Before using the page,you need to call dev.setServerAddress() manually and
   * then login() again, since setServerAddress logs out as a side effect.
   */
  backgroundApiUser: [
    async ({ backgroundApiAuth }, use) => {
      const backgroundApiUser = new ApiUserFixture(backgroundApiAuth);
      await use(backgroundApiUser);
    },
    { scope: 'worker' },
  ],
  dev: async ({ page }, use) => {
    const dev = new DevFixture(page);
    await use(dev);
  },
  /**
   * Creates an admin and logs in as that admin for the duraction of the test suite
   */
  newAdmin: [
    async ({ backgroundApiUser }, use, workerInfo) => {
      const adminUser = await backgroundApiUser.createUser(
        backgroundApiUser.new({
          userProps: {
            user: {
              active: true,
              admin: true,
              username: safeName(`test-admin-${workerInfo.workerIndex}`),
            },
          },
        }),
      );
      await backgroundApiUser.apiAuth.login({
        creds: { password: adminUser.password!, username: adminUser.user!.username },
      });
      await use(adminUser);
      await backgroundApiUser.apiAuth.login();
      await backgroundApiUser.patchUser(adminUser.user!.id!, { active: false });
    },
    { scope: 'worker' },
  ],
  user: async ({ page }, use) => {
    const user = new UserFixture(page);
    await use(user);
  },
});
