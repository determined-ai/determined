import { expect } from '@playwright/test';

import { test } from 'e2e/fixtures/global-fixtures';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';

test.describe('Experiement List', () => {
  test.beforeEach(async ({ auth, dev }) => {
    await dev.setServerAddress();
    await auth.login();
  });

  test('Navigate to Experiment List', async ({ page }) => {
    const projectDetailsPage = new ProjectDetails(page);
    await projectDetailsPage.gotoProject();
    await expect(page).toHaveTitle(projectDetailsPage.title);
  });

  test.skip('Click around the data grid', async ({ page }) => {
    // This test expects a project to have been deployed.
    // This test.skip is useful to show an example of what tests can do
    const projectDetailsPage = new ProjectDetails(page);
    await projectDetailsPage.gotoProject();
    await expect(projectDetailsPage.f_experiemntList.dataGrid.rows.pwLocator).toHaveCount(1);
    await projectDetailsPage.f_experiemntList.dataGrid.setColumnHeight();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.setColumnDefs();
    const row = await projectDetailsPage.f_experiemntList.dataGrid.getRowByColumnValue(
      'Trials',
      '1',
    );
    await row.clickSelect();
    expect(await row.isSelected()).toBeTruthy();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.clickSelectDropdown();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.selectDropdown.select5.pwLocator.click();
  });
});
