import { test as base } from '@playwright/test';

import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { DevFixture } from 'e2e/fixtures/dev.fixture';

type CustomFixtures = {
  dev: DevFixture;
  auth: AuthFixture;
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
});
