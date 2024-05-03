import { test } from 'e2e/fixtures/global-fixtures';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';

import { expect } from '@playwright/test';

test.describe('Experiement List', () => {

  test('Navigate to Experiment List', async ({ authedPage }) => {
    const projectDetailsPage = new ProjectDetails(authedPage);
    await projectDetailsPage.gotoProject();
    await expect(authedPage).toHaveTitle(projectDetailsPage.title);
    await expect(projectDetailsPage.f_experiemntList.tableActionBar.pwLocator).toBeVisible();
  });

  test.skip('Click around the data grid', async ({ authedPage }) => {
    // This test expects a project to have been deployed.
    // This test.skip is useful to show an example of what tests can do
    const projectDetailsPage = new ProjectDetails(authedPage);
    await projectDetailsPage.gotoProject();
    await expect(projectDetailsPage.f_experiemntList.dataGrid.rows.pwLocator).toHaveCount(1);
    await projectDetailsPage.f_experiemntList.dataGrid.setColumnHeight();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.setColumnDefs();
    const row = await projectDetailsPage.f_experiemntList.dataGrid.getRowByColumnValue(
      'Trials',
      '1',
    );
    await row.clickColumn('Select');
    expect(await row.isSelected()).toBeTruthy();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.clickSelectDropdown();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.selectDropdown.select5.pwLocator.click();
    await row.clickColumn('ID');
    await authedPage.waitForURL(/overview/);
  });
});
