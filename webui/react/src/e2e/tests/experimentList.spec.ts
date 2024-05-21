import { expect } from '@playwright/test';

import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { test } from 'e2e/fixtures/global-fixtures';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';
import { detExecSync, fullPath } from 'e2e/utils/detCLI';

test.describe('Experiement List', () => {
  let projectDetailsPage: ProjectDetails;

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

  test.beforeEach(async ({ authedPage }) => {
    projectDetailsPage = new ProjectDetails(authedPage);
    await projectDetailsPage.gotoProject();
    await expect(projectDetailsPage.f_experiemntList.dataGrid.rows.pwLocator).not.toHaveCount(0);
    await projectDetailsPage.f_experiemntList.dataGrid.setColumnHeight();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.setColumnDefs();
  });

  test('Navigate to Experiment List', async ({ authedPage }) => {
    await projectDetailsPage.gotoProject();
    await expect(authedPage).toHaveTitle(projectDetailsPage.title);
    await expect(projectDetailsPage.f_experiemntList.tableActionBar.pwLocator).toBeVisible();
  });

  test('Column Picker', async () => {
    const columnTitle = 'Forked From',
      columnTestid = 'forkedFrom';
    const columnPicker =
      await projectDetailsPage.f_experiemntList.tableActionBar.columnPickerMenu.open();
    const checkbox = columnPicker.columnPickerTab.columns.listItem(columnTestid).checkbox;
    const grid = projectDetailsPage.f_experiemntList.dataGrid;
    // close the popover with a click elsewhere
    const closePopover = async () =>
      await projectDetailsPage.f_experiemntList.tableActionBar.expNum.pwLocator.click();
    // trial click will wait for the element to be stable
    const waitTableStable = async () => await grid.pwLocator.click({ trial: true });

    await test.step('Uncheck as a part of test setup', async () => {
      // "Forked is not enabled by default. If we're on a dirty setup, we need to disable it first."
      await checkbox.pwLocator.uncheck();
      await closePopover();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect(grid.headRow.columnDefs.get(columnTitle)).toBeUndefined();
    });

    await test.step('Check', async () => {
      await columnPicker.open();
      await checkbox.pwLocator.check();
      await closePopover();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect(grid.headRow.columnDefs.get(columnTitle)).toBeTruthy();
      expect(await grid.scrollColumnIntoViewByName(columnTitle)).toBe(true);
    });

    await test.step('Uncheck', async () => {
      await columnPicker.open();
      await checkbox.pwLocator.uncheck();
      await closePopover();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect(grid.headRow.columnDefs.get(columnTitle)).toBeUndefined();
    });
  });

  test('Click around the data grid', async ({ authedPage }) => {
    const row = await projectDetailsPage.f_experiemntList.dataGrid.getRowByColumnValue('ID', '1');
    await row.clickColumn('Select');
    expect(await row.isSelected()).toBeTruthy();
    await expect((await row.getCellByColumnName('Checkpoints')).pwLocator).toHaveText('0');
    await (
      await projectDetailsPage.f_experiemntList.dataGrid.headRow.selectDropdown.open()
    ).select5.pwLocator.click();
    await row.clickColumn('ID');
    await authedPage.waitForURL(/overview/);
  });
});
