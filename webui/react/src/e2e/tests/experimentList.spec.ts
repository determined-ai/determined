import { expect } from '@playwright/test';

import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { test } from 'e2e/fixtures/global-fixtures';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';
import { detExecSync, fullPath } from 'e2e/utils/detCLI';

test.describe('Experiement List', () => {
  let projectDetailsPage: ProjectDetails;

  // close the popover with a click elsewhere
  const closePopover = async () =>
    await projectDetailsPage.f_experiemntList.tableActionBar.expNum.pwLocator.click();
  // trial click will wait for the element to be stable
  const waitTableStable = async () =>
    await projectDetailsPage.f_experiemntList.dataGrid.pwLocator.click({ trial: true });
  const getExpNum = async () => {
    const expNum =
      await projectDetailsPage.f_experiemntList.tableActionBar.expNum.pwLocator.textContent();
    if (expNum === null) throw new Error('Experiment number is null');
    return parseInt(expNum);
  };

  test.beforeAll(async ({ browser }) => {
    const pageSetupTeardown = await browser.newPage();
    const authFixtureSetupTeardown = new AuthFixture(pageSetupTeardown);
    const projectDetailsPageSetupTeardown = new ProjectDetails(pageSetupTeardown);
    await authFixtureSetupTeardown.login();
    await projectDetailsPageSetupTeardown.gotoProject();
    await test.step('Create an experiment if not already present', async () => {
      await projectDetailsPageSetupTeardown.f_experiemntList.tableActionBar.pwLocator.waitFor();
      await expect(
        projectDetailsPageSetupTeardown.f_experiemntList.tableActionBar.expNum.pwLocator,
      ).toContainText('experiment');
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
    const grid = projectDetailsPage.f_experiemntList.dataGrid;

    await projectDetailsPage.gotoProject();
    await expect(projectDetailsPage.f_experiemntList.dataGrid.rows.pwLocator).not.toHaveCount(0);
    await test.step('Reset Columns', async () => {
      const columnPicker =
        await projectDetailsPage.f_experiemntList.tableActionBar.columnPickerMenu.open();
      await columnPicker.columnPickerTab.reset.pwLocator.click();
      await closePopover();
      await waitTableStable();
    });
    await test.step('Reset Filters', async () => {
      const tableFilter =
        await projectDetailsPage.f_experiemntList.tableActionBar.tableFilter.open();
      await tableFilter.filterForm.clearFilters.pwLocator.click();
      await closePopover();
      await waitTableStable();
    });
    await grid.headRow.setColumnDefs();
    await projectDetailsPage.f_experiemntList.dataGrid.setColumnHeight();
    await projectDetailsPage.f_experiemntList.dataGrid.headRow.setColumnDefs();
  });

  test('Navigate to Experiment List', async ({ authedPage }) => {
    await projectDetailsPage.gotoProject();
    await expect(authedPage).toHaveTitle(projectDetailsPage.title);
    await expect(projectDetailsPage.f_experiemntList.tableActionBar.pwLocator).toBeVisible();
  });

  test('Column Picker add and remove', async () => {
    const columnTitle = 'Forked From',
      columnTestid = 'forkedFrom';
    const columnPicker = projectDetailsPage.f_experiemntList.tableActionBar.columnPickerMenu;
    const checkbox = columnPicker.columnPickerTab.columns.listItem(columnTestid).checkbox;
    const grid = projectDetailsPage.f_experiemntList.dataGrid;

    await test.step('Check', async () => {
      await columnPicker.open();
      await checkbox.pwLocator.check();
      await closePopover();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect(grid.headRow.columnDefs.get(columnTitle)).toBeTruthy();
      await grid.scrollColumnIntoViewByName(columnTitle);
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

  test('Column Picker Show All and Hide All', async () => {
    const columnPicker = projectDetailsPage.f_experiemntList.tableActionBar.columnPickerMenu;
    const grid = projectDetailsPage.f_experiemntList.dataGrid;
    let previousTabs = grid.headRow.columnDefs.size;

    await test.step('General Tab Show', async () => {
      await columnPicker.open();
      await columnPicker.columnPickerTab.showAll.pwLocator.click();
      await closePopover();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect.soft(previousTabs).toBeLessThan(grid.headRow.columnDefs.size);
      previousTabs = grid.headRow.columnDefs.size;
    });

    await test.step('Hyperparameter Tab Show', async () => {
      await columnPicker.open();
      await columnPicker.hyperparameterTab.pwLocator.click();
      await columnPicker.columnPickerTab.showAll.pwLocator.click();
      await closePopover();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect.soft(previousTabs).toBeLessThan(grid.headRow.columnDefs.size);
      previousTabs = grid.headRow.columnDefs.size;
    });

    await test.step('General Tab Hide', async () => {
      await columnPicker.open();
      await columnPicker.generalTab.pwLocator.click();
      await columnPicker.columnPickerTab.showAll.pwLocator.click();
      await closePopover();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect.soft(previousTabs).toBeGreaterThan(grid.headRow.columnDefs.size);
      previousTabs = grid.headRow.columnDefs.size;
    });

    await test.step('General Search and Show', async () => {
      const columnTitle = 'ID',
        idColumns = 3;
      await columnPicker.open();
      await columnPicker.columnPickerTab.search.pwLocator.fill(columnTitle);
      await columnPicker.columnPickerTab.showAll.pwLocator.click();
      await closePopover();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect.soft(previousTabs + idColumns).toBeLessThanOrEqual(grid.headRow.columnDefs.size);
      expect(grid.headRow.columnDefs.get(columnTitle)).toBeTruthy();
      await grid.scrollColumnIntoViewByName(columnTitle);
    });
  });

  test('Table Filter', async () => {
    const tableFilter = projectDetailsPage.f_experiemntList.tableActionBar.tableFilter;
    const totalExperiments = await getExpNum();

    const filterScenario = async (
      name: string,
      scenario: () => Promise<void>,
      expectedValue: number,
    ) => {
      await test.step(name, async () => {
        await tableFilter.open();
        await scenario();
        await waitTableStable();
        await expect.poll(async () => await getExpNum()).toBe(expectedValue);
        await closePopover();
      });
    };

    await filterScenario(
      'Filter with ID',
      async () => {
        await tableFilter.filterForm.filter.filterFields.columnName.selectMenuOption('ID');
        await expect(tableFilter.filterForm.filter.filterFields.operator.pwLocator).toHaveText('=');
        await tableFilter.filterForm.filter.filterFields.operator.selectMenuOption('=');
        await tableFilter.filterForm.filter.filterFields.valueNumber.pwLocator.fill('1');
      },
      1,
    );

    await filterScenario(
      'Filter against ID',
      async () => {
        await tableFilter.filterForm.filter.filterFields.operator.selectMenuOption('!=');
      },
      totalExperiments - 1,
    );

    await filterScenario(
      'Filter OR',
      async () => {
        // This looks a little screwy with nth(1) in some places. Everything here is referring to the second filterfield row.
        // [INFENG-715]
        await tableFilter.filterForm.addCondition.pwLocator.click();
        await tableFilter.filterForm.filter.filterFields.conjunctionContainer.conjunctionSelect.pwLocator.click();
        await tableFilter.filterForm.filter.filterFields.conjunctionContainer.conjunctionSelect._menu.pwLocator.waitFor();
        await tableFilter.filterForm.filter.filterFields.conjunctionContainer.conjunctionSelect.selectMenuOption(
          'or',
        );
        await tableFilter.filterForm.filter.filterFields.columnName.pwLocator.nth(1).click();
        await tableFilter.filterForm.filter.filterFields.columnName._menu.pwLocator.waitFor();
        await tableFilter.filterForm.filter.filterFields.columnName.selectMenuOption('ID');
        await expect(
          tableFilter.filterForm.filter.filterFields.operator.pwLocator.nth(1),
        ).toHaveText('=');
        await tableFilter.filterForm.filter.filterFields.operator.pwLocator.nth(1).click();
        await tableFilter.filterForm.filter.filterFields.operator._menu.pwLocator.waitFor();
        await tableFilter.filterForm.filter.filterFields.operator.selectMenuOption('=');
        await tableFilter.filterForm.filter.filterFields.valueNumber.pwLocator.nth(1).fill('1');
      },
      totalExperiments,
    );
  });

  test('Click around the data grid', async ({ authedPage }) => {
    const row = await projectDetailsPage.f_experiemntList.dataGrid.getRowByColumnValue('ID', '1');
    await row.clickColumn('Select');
    expect(await row.isSelected()).toBeTruthy();
    await expect((await row.getCellByColumnName('Checkpoints')).pwLocator).toHaveText('0');
    await (
      await projectDetailsPage.f_experiemntList.dataGrid.headRow.selectDropdown.open()
    ).select5.pwLocator.click();
    await projectDetailsPage.f_experiemntList.dataGrid.scrollLeft();
    await row.clickColumn('ID');
    await authedPage.waitForURL(/overview/);
  });
});
