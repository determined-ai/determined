import { expect } from '@playwright/test';

import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { test } from 'e2e/fixtures/global-fixtures';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';
import { detExecSync, fullPath } from 'e2e/utils/detCLI';

test.describe('Experiement List', () => {
  test.beforeAll(async ({ browser }) => {
    const pageSetupTeardown = await browser.newPage();
    const authFixtureSetupTeardown = new AuthFixture(pageSetupTeardown);
    const projectDetailsPageSetupTeardown = new ProjectDetails(pageSetupTeardown);
    await authFixtureSetupTeardown.login();
    await projectDetailsPageSetupTeardown.gotoProject();
    await test.step('Create an experiment if not already present', async () => {
      await expect(
        projectDetailsPageSetupTeardown.f_experiemntList.tableActionBar.pwLocator,
      ).toBeVisible();
      // wait for it to not say "loading experiments..."

      if (
        await projectDetailsPageSetupTeardown.f_experiemntList.noExperimentsMessage.pwLocator.isVisible()
      ) {
        detExecSync(
          `experiment create ${fullPath(
            '/../../examples/tutorials/mnist_pytorch/const.yaml',
          )} --paused`,
        );
        await pageSetupTeardown.reload();
        await expect(
          projectDetailsPageSetupTeardown.f_experiemntList.dataGrid.rows.pwLocator,
        ).not.toHaveCount(0);
      }
    });
    await authFixtureSetupTeardown.logout();
    await pageSetupTeardown.close();
  });

  test('Navigate to Experiment List', async ({ authedPage }) => {
    const projectDetailsPage = new ProjectDetails(authedPage);
    await projectDetailsPage.gotoProject();
    await expect(authedPage).toHaveTitle(projectDetailsPage.title);
    await expect(projectDetailsPage.f_experiemntList.tableActionBar.pwLocator).toBeVisible();
  });

  test('Click around the data grid', async ({ authedPage }) => {
    const projectDetailsPage = new ProjectDetails(authedPage);
    await projectDetailsPage.gotoProject();
    await expect(projectDetailsPage.f_experiemntList.dataGrid.rows.pwLocator).not.toHaveCount(0);
    await projectDetailsPage.f_experiemntList.dataGrid.setColumnHeight();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.setColumnDefs();
    const row = await projectDetailsPage.f_experiemntList.dataGrid.getRowByColumnValue('ID', '1');
    await row.clickColumn('Select');
    expect(await row.isSelected()).toBeTruthy();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.selectDropdown.open();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.selectDropdown.select5.pwLocator.click();
    await row.clickColumn('ID');
    await authedPage.waitForURL(/overview/);
  });
});
