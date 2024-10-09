import { Page } from '@playwright/test';

import { apiUrl } from 'e2e/utils/envVars';

export class DevFixture {
  setServerAddress = async (page: Page): Promise<void> => {
    // Tells the frontend where to find the backend if built for a different url.
    // Incidentally reloads and logs out of Determined.
    await page.goto('/');
    await page.evaluate(`dev.setServerAddress("${apiUrl()}")`);
    await page.reload();
    // dev.setServerAddress fires a logout request in the background, so we will wait until no network traffic is happening.
    await page.waitForLoadState('networkidle');
  };
}
