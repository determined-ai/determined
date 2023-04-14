import { test as base } from '@playwright/test';

import { DevFixture } from './fixtures/dev.fixture';

type CustomFixtures = {
  dev: DevFixture;
};

// https://playwright.dev/docs/test-fixtures
export const test = base.extend<CustomFixtures>({
  dev: async ({ page }, use) => {
    const dev = new DevFixture(page);
    await dev.setServerAddress();
    await use(dev);
  },
});
