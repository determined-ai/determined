import { execSync } from 'child_process';
import path from 'path';

import { expect } from '@playwright/test';

import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { test } from 'e2e/fixtures/global-fixtures';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';

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
        const experimentPath = path.join(
          process.cwd(),
          '/../../examples/tutorials/mnist_pytorch/const.yaml',
        );
        const detCommandBase = `${process.env.PW_DET_PATH} -m ${process.env.PW_DET_MASTER}`;
        execSync(`${detCommandBase} user logout`);
        execSync(
          `echo ${process.env.PW_PASSWORD} | ${detCommandBase} user login ${process.env.PW_USER_NAME}`,
          { stdio: 'inherit' },
        );
        execSync(`${detCommandBase} experiment create ${experimentPath} --paused`);
        await pageSetupTeardown.reload();
        await expect(
          projectDetailsPageSetupTeardown.f_experiemntList.dataGrid.rows.pwLocator,
        ).toHaveCount(1);
      }
    });
    await authFixtureSetupTeardown.logout();
    await pageSetupTeardown.close();
  });

  test.beforeEach(async ({ auth, dev }) => {
    await dev.setServerAddress();
    await auth.login();
  });

  test('Navigate to Experiment List', async ({ page }) => {
    const projectDetailsPage = new ProjectDetails(page);
    await projectDetailsPage.gotoProject();
    await expect(page).toHaveTitle(projectDetailsPage.title);
    await expect(projectDetailsPage.f_experiemntList.tableActionBar.pwLocator).toBeVisible();
  });

  test('Click around the data grid', async ({ page }) => {
    const projectDetailsPage = new ProjectDetails(page);
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
    await page.waitForURL(/overview/);
  });
});
