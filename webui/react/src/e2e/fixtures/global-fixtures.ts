import { test as base, Page } from '@playwright/test';

import { ApiAuthFixture } from './api.auth.fixture';
import { AuthFixture } from './auth.fixture';
import { DevFixture } from './dev.fixture';
import { UserFixture } from './user.fixture';

type CustomFixtures = {
  dev: DevFixture;
  auth: AuthFixture;
  apiAuth: ApiAuthFixture;
  user: UserFixture;
  authedPage: Page;
};

// https://playwright.dev/docs/test-fixtures
export const test = base.extend<CustomFixtures>({
  // get the auth but allow yourself to log in through the api manually.
apiAuth: async ({ playwright, browser }, use) => {
    const apiAuth = new ApiAuthFixture(playwright.request, browser);
    await use(apiAuth);
  },

  auth: async ({ page }, use) => {
    const auth = new AuthFixture(page);
    await use(auth);
  },

  // get a page already logged in
authedPage: async ({ playwright, browser, dev }, use) => {
    await dev.setServerAddress();
    const apiAuth = new ApiAuthFixture(playwright.request, browser, dev.page);
    await apiAuth.login();
    await use(apiAuth.page);
  },

dev: async ({ page }, use) => {
    const dev = new DevFixture(page);
    await use(dev);
  },

  user: async ({ page }, use) => {
    const user = new UserFixture(page);
    await use(user);
  },
});
