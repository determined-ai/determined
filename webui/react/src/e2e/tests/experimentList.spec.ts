import { AuthFixture } from 'e2e/fixtures/auth.fixture';
import { expect, test } from 'e2e/fixtures/global-fixtures';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';
import { detExecSync, fullPath } from 'e2e/utils/detCLI';
import { safeName } from 'e2e/utils/naming';

test.describe('Experiment List', () => {
  let projectDetailsPage: ProjectDetails;
  // trial click to wait for the element to be stable won't work here
  const waitTableStable = async () => await projectDetailsPage._page.waitForTimeout(2_000);
  const getExpNum = async () => {
    const expNum =
      await projectDetailsPage.f_experimentList.tableActionBar.expNum.pwLocator.textContent();
    if (expNum === null) throw new Error('Experiment number is null');
    return parseInt(expNum);
  };

  test.beforeAll(async ({ browser, dev }) => {
    const pageSetupTeardown = await browser.newPage();
    await dev.setServerAddress(pageSetupTeardown);
    const authFixtureSetupTeardown = new AuthFixture(pageSetupTeardown);
    const projectDetailsPageSetupTeardown = new ProjectDetails(pageSetupTeardown);
    await authFixtureSetupTeardown.login();
    await projectDetailsPageSetupTeardown.gotoProject();
    await test.step('Create an experiment if not already present', async () => {
      await projectDetailsPageSetupTeardown.f_experimentList.tableActionBar.pwLocator.waitFor();
      await expect(
        projectDetailsPageSetupTeardown.f_experimentList.tableActionBar.expNum.pwLocator,
      ).toContainText('experiment');
      if (
        await projectDetailsPageSetupTeardown.f_experimentList.noExperimentsMessage.pwLocator.isVisible()
      ) {
        detExecSync(
          `experiment create ${fullPath(
            '/../../examples/tutorials/mnist_pytorch/const.yaml',
          )} --paused`,
        );
        await pageSetupTeardown.reload();
        await expect(
          projectDetailsPageSetupTeardown.f_experimentList.dataGrid.rows.pwLocator,
        ).not.toHaveCount(0);
      }
    });
    await authFixtureSetupTeardown.logout();
    await pageSetupTeardown.close();
  });

  test.beforeEach(async ({ authedPage }) => {
    projectDetailsPage = new ProjectDetails(authedPage);
    const grid = projectDetailsPage.f_experimentList.dataGrid;

    await projectDetailsPage.gotoProject();
    await expect(projectDetailsPage.f_experimentList.dataGrid.rows.pwLocator).not.toHaveCount(0, {
      timeout: 10_000,
    });
    await test.step('Deselect', async () => {
      try {
        await grid.headRow.selectDropdown.menuItem('select-none').select({ timeout: 1_000 });
      } catch (e) {
        // close the dropdown by clicking elsewhere
        await projectDetailsPage.f_experimentList.tableActionBar.expNum.pwLocator.click();
      }
    });
    await test.step('Reset Columns', async () => {
      const columnPicker =
        await projectDetailsPage.f_experimentList.tableActionBar.columnPickerMenu.open();
      await columnPicker.columnPickerTab.reset.pwLocator.click();
      await columnPicker.close();
    });
    await test.step('Reset Filters', async () => {
      const tableFilter =
        await projectDetailsPage.f_experimentList.tableActionBar.tableFilter.open();
      await tableFilter.filterForm.clearFilters.pwLocator.click();
      await tableFilter.close();
    });
    await waitTableStable();
    await grid.setColumnHeight();
    await grid.headRow.setColumnDefs();
  });

  test.skip('Column Picker Check and Uncheck', async () => {
    // BUG [ET-287]
    const columnTitle = 'Forked From',
      columnTestid = 'forkedFrom';
    const columnPicker = projectDetailsPage.f_experimentList.tableActionBar.columnPickerMenu;
    const checkbox = columnPicker.columnPickerTab.columns.listItem(columnTestid).checkbox;
    const grid = projectDetailsPage.f_experimentList.dataGrid;

    await test.step('Check', async () => {
      await columnPicker.open();
      await checkbox.pwLocator.check();
      await columnPicker.close();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect(grid.headRow.columnDefs.get(columnTitle)).toBeTruthy();
      await grid.scrollColumnIntoViewByName(columnTitle);
    });

    await test.step('Uncheck', async () => {
      await columnPicker.open();
      await checkbox.pwLocator.uncheck();
      await columnPicker.close();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect(grid.headRow.columnDefs.get(columnTitle)).toBeUndefined();
    });
  });

  test('Column Picker Show All and Hide All', async () => {
    test.slow();
    const columnPicker = projectDetailsPage.f_experimentList.tableActionBar.columnPickerMenu;
    const grid = projectDetailsPage.f_experimentList.dataGrid;
    let previousTabs = grid.headRow.columnDefs.size;

    await test.step('General Tab Show All', async () => {
      await columnPicker.open();
      await columnPicker.columnPickerTab.showAll.pwLocator.click();
      await columnPicker.close();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect.soft(previousTabs).toBeLessThan(grid.headRow.columnDefs.size);
      previousTabs = grid.headRow.columnDefs.size;
    });

    await test.step('Hyperparameter Tab Show All', async () => {
      await columnPicker.open();
      await columnPicker.hyperparameterTab.pwLocator.click();
      await columnPicker.columnPickerTab.showAll.pwLocator.click();
      await columnPicker.close();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect.soft(previousTabs).toBeLessThan(grid.headRow.columnDefs.size);
      previousTabs = grid.headRow.columnDefs.size;
    });

    await test.step('General Tab Hide All', async () => {
      await columnPicker.open();
      await columnPicker.generalTab.pwLocator.click();
      await expect.soft(columnPicker.columnPickerTab.showAll.pwLocator).toHaveText('Hide all');
      await columnPicker.columnPickerTab.showAll.pwLocator.click();
      await columnPicker.close();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect.soft(previousTabs).toBeGreaterThan(grid.headRow.columnDefs.size);
      previousTabs = grid.headRow.columnDefs.size;
    });

    await test.step('General Search[ID] and Show All', async () => {
      const columnTitle = 'ID',
        idColumns = 3;
      await columnPicker.open();
      await columnPicker.columnPickerTab.search.pwLocator.fill(columnTitle);
      await columnPicker.columnPickerTab.showAll.pwLocator.click();
      await columnPicker.close();
      await waitTableStable();
      await grid.headRow.setColumnDefs();
      expect.soft(previousTabs + idColumns).toBeLessThanOrEqual(grid.headRow.columnDefs.size);
      expect(grid.headRow.columnDefs.get(columnTitle)).toBeTruthy();
      await grid.scrollColumnIntoViewByName(columnTitle);
    });
  });

  test('Table Filter', async () => {
    test.slow();
    const tableFilter = projectDetailsPage.f_experimentList.tableActionBar.tableFilter;
    const totalExperiments = await getExpNum();

    const filterScenario = async (
      name: string,
      scenario: () => Promise<void>,
      expectedValue: number,
    ) => {
      await test.step(name, async () => {
        await tableFilter.open();
        await scenario();
        // [ET-284] - Sometimes, closing the popover too quickly causes the filter to not apply.
        await waitTableStable();
        await expect.poll(async () => await getExpNum()).toBe(expectedValue);
        await tableFilter.close();
      });
    };

    await filterScenario(
      'Filter With ID',
      async () => {
        await tableFilter.filterForm.filter.filterFields.columnName.selectMenuOption('ID');
        await expect(tableFilter.filterForm.filter.filterFields.operator.pwLocator).toHaveText('=');
        await tableFilter.filterForm.filter.filterFields.operator.selectMenuOption('=');
        await tableFilter.filterForm.filter.filterFields.valueNumber.pwLocator.fill('1');
      },
      1,
    );

    await filterScenario(
      'Filter Against ID',
      async () => {
        await expect(
          tableFilter.filterForm.filter.filterFields.columnName.selectionItem.pwLocator,
        ).toHaveText('ID');
        await tableFilter.filterForm.filter.filterFields.operator.selectMenuOption('!=');
      },
      totalExperiments - 1,
    );

    await filterScenario(
      'Filter OR',
      async () => {
        await tableFilter.filterForm.addCondition.pwLocator.click();
        const secondFilterField = tableFilter.filterForm.filter.filterFields.nth(1);
        const conjunction = secondFilterField.conjunctionContainer.conjunctionSelect;
        await conjunction.pwLocator.click();
        await conjunction._menu.pwLocator.waitFor();
        await conjunction.menuItem('or').pwLocator.click();
        await conjunction._menu.pwLocator.waitFor({ state: 'hidden' });

        const columnName = secondFilterField.columnName;
        await columnName.pwLocator.click();
        await columnName._menu.pwLocator.waitFor();
        await columnName.menuItem('ID').pwLocator.click();
        await columnName._menu.pwLocator.waitFor({ state: 'hidden' });

        const operator = secondFilterField.operator;
        await expect(operator.pwLocator).toHaveText('=');
        await operator.pwLocator.click();
        await operator._menu.pwLocator.waitFor();
        await operator.menuItem('=').pwLocator.click();
        await operator._menu.pwLocator.waitFor({ state: 'hidden' });

        await secondFilterField.valueNumber.pwLocator.fill('1');
      },
      totalExperiments,
    );
  });

  test('Datagrid Functionality Validations', async ({ authedPage }) => {
    const row = await projectDetailsPage.f_experimentList.dataGrid.getRowByIndex(0);
    await test.step('Select Row', async () => {
      await row.clickColumn('Select');
      expect.soft(await row.isSelected()).toBeTruthy();
    });
    await test.step('Read Cell Value', async () => {
      await expect.soft((await row.getCellByColumnName('ID')).pwLocator).toHaveText(/\d+/);
    });
    await test.step('Select 5', async () => {
      await (
        await projectDetailsPage.f_experimentList.dataGrid.headRow.selectDropdown.open()
      ).select5.pwLocator.click();
    });
    await test.step('Experiment Overview Navigation', async () => {
      await projectDetailsPage.f_experimentList.dataGrid.scrollLeft();
      const textContent = await (await row.getCellByColumnName('ID')).pwLocator.textContent();
      await row.clickColumn('ID');
      if (textContent === null) throw new Error('Cannot read row id');
      await authedPage.waitForURL(new RegExp(textContent));
    });
  });

  test('Datagrid Actions', async () => {
    const row = await projectDetailsPage.f_experimentList.dataGrid.getRowByIndex(0);
    await row.experimentActionDropdown.open();
    // feel free to split actions into their own test cases. this is just a starting point
    await test.step('Edit', async () => {
      const editedValue = safeName('EDITED_EXPERIMENT_NAME');
      await row.experimentActionDropdown.edit.pwLocator.click();
      await row.experimentActionDropdown.editModal.nameInput.pwLocator.fill(editedValue);
      await row.experimentActionDropdown.editModal.footer.submit.pwLocator.click();
      await expect.soft((await row.getCellByColumnName('Name')).pwLocator).toHaveText(editedValue);
    });
    // await test.step('Pause', async () => {
    //   // what happens if the experiment is already paused?
    // });
    // await test.step('Stop', async () => {
    //   // what happens if the experiment is already stopped?
    // });
    // await test.step('Kill', async () => {
    //   // what happens if the experiment is already killed? do we need to change beforeAll logic?
    // });
    // await test.step('Move', async () => {
    //   // move to where? do we need a new project? check project spec
    // });
    // await test.step('Archive / Unarchive', async () => {
    //   // what happens if the experiment is already archived?
    // });
    // await test.step('View in Tensorboard', async () => {
    //   // might want something like this
    //   // await authedPage.waitForURL(;
    // });
    // await test.step('Hyperparameter Search', async () => {});
  });
});
