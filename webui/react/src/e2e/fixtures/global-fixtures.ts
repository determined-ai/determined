import { test as base } from '@playwright/test';

import { AuthFixture } from './auth.fixture';
import { DevFixture } from './dev.fixture';
import { UserFixture } from './user.fixture';

type CustomFixtures = {
  dev: DevFixture;
  auth: AuthFixture;
  user: UserFixture;
};

// https://playwright.dev/docs/test-fixtures
export const test = base.extend<CustomFixtures>({
  auth: async ({ page }, use) => {
    const auth = new AuthFixture(page);
    await use(auth);
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
