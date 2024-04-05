import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { ExperimentList } from 'e2e/models/pages/ExperimentList';

test.describe('Experiement List', () => {
  test.beforeEach(async ({ auth, dev }) => {
    await dev.setServerAddress();
    await auth.login();
  });

  test('Navigate to User Management', async ({ page }) => {
    const experiementListPage = new ExperimentList(page);
    await experiementListPage.gotoProject();
    await expect(page).toHaveTitle(experiementListPage.title);

  });
});
