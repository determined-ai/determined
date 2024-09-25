import { expect, test } from 'e2e/fixtures/global-fixtures';
import { ProjectDetails } from 'e2e/models/pages/ProjectDetails';
import { detExecSync, fullPath } from 'e2e/utils/detCLI';
import { safeName } from 'e2e/utils/naming';
import { repeatWithFallback } from 'e2e/utils/polling';
import { ExperimentBase } from 'types';

test.describe('Experiment List', () => {
  let projectDetailsPage: ProjectDetails;
  // trial click to wait for the element to be stable won't work here
  const waitTableStable = async () => await projectDetailsPage._page.waitForTimeout(2_000);
  const getCount = async () => {
    const count =
      await projectDetailsPage.f_experimentList.tableActionBar.count.pwLocator.textContent();
    if (count === null) throw new Error('Count is null');
    return parseInt(count);
  };

  test.beforeAll(async ({ backgroundAuthedPage, newWorkspace, newProject }) => {
    const projectDetailsPageSetup = new ProjectDetails(backgroundAuthedPage);
    await projectDetailsPageSetup.gotoProject(newProject.response.project.id);
    await test.step('Create an experiment', async () => {
      await expect(
        projectDetailsPageSetup.f_experimentList.tableActionBar.count.pwLocator,
      ).toContainText('experiment');
      Array(4)
        .fill(null)
        .forEach(() => {
          detExecSync(
            `experiment create ${fullPath('examples/tutorials/mnist_pytorch/adaptive.yaml')} --paused --project_id ${newProject.response.project.id}`,
          );
        });

      const experiments: ExperimentBase[] = JSON.parse(
        detExecSync(
          `project list-experiments --json ${newWorkspace.response.workspace.name} ${newProject.response.project.name}`,
        ),
      );
      detExecSync(`experiment kill ${experiments[experiments.length - 1]?.id}`); // Experiments must be in terminal state to archive
      detExecSync(`experiment archive ${experiments[experiments.length - 1]?.id}`);

      await expect(
        projectDetailsPageSetup.f_experimentList.dataGrid.rows.pwLocator,
      ).not.toHaveCount(0, { timeout: 10_000 });
    });
  });

  test.beforeEach(async ({ authedPage, newProject }) => {
    test.slow();
    projectDetailsPage = new ProjectDetails(authedPage);
    const grid = projectDetailsPage.f_experimentList.dataGrid;

    await projectDetailsPage.gotoProject(newProject.response.project.id);
    await expect(projectDetailsPage.f_experimentList.dataGrid.rows.pwLocator).not.toHaveCount(0, {
      timeout: 10_000,
    });
    await test.step('Deselect', async () => {
      try {
        await grid.headRow.selectDropdown.menuItem('select-none').select({ timeout: 1_000 });
      } catch (e) {
        // close the dropdown by clicking elsewhere
        await projectDetailsPage.f_experimentList.tableActionBar.count.pwLocator.click();
      }
    });
    await test.step('Reset Columns', async () => {
      const columnPicker =
        await projectDetailsPage.f_experimentList.tableActionBar.columnPickerMenu.open();
      await waitTableStable();
      await columnPicker.columnPickerTab.reset.pwLocator.click();
      await columnPicker.close();
      await waitTableStable();
    });
    await test.step('Sort Oldest → Newest', async () => {
      // reset
      const sortContent =
        await projectDetailsPage.f_experimentList.tableActionBar.multiSortMenu.open();
      await sortContent.multiSort.reset.pwLocator.click();
      // the menu doesn't close in local automation, but it works with mouse events
      // manually and sometimes on ci. let's just close it manually
      await sortContent.close();
      await sortContent.open();
      // set sort
      const firstRow = sortContent.multiSort.rows.nth(0);
      await firstRow.column.selectMenuOption('Start time');
      await firstRow.order.selectMenuOption('Oldest → Newest');
      await sortContent.close();
      await waitTableStable();
    });
    await test.step('Reset Filters', async () => {
      const tableFilter =
        await projectDetailsPage.f_experimentList.tableActionBar.tableFilter.open();
      await tableFilter.filterForm.clearFilters.pwLocator.click();
      await tableFilter.close();
      await waitTableStable();
    });
    await test.step('Reset Show Archived', async () => {
      const tableFilter =
        await projectDetailsPage.f_experimentList.tableActionBar.tableFilter.open();
      await expect(
        repeatWithFallback(
          async () =>
            await expect(tableFilter.filterForm.showArchived.pwLocator).toHaveAttribute(
              'aria-checked',
              'false',
            ),
          async () => await tableFilter.filterForm.showArchived.pwLocator.click(),
        ),
      ).toPass({ timeout: 30_000 });
      await tableFilter.close();
      await waitTableStable();
    });
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
    const tableFilter = projectDetailsPage.f_experimentList.tableActionBar.tableFilter;
    const totalExperiments = await getCount();

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
        await expect.poll(async () => await getCount()).toBe(expectedValue);
        await tableFilter.close();
      });
    };

    const row = projectDetailsPage.f_experimentList.dataGrid.getRowByIndex(0);
    const id = await (await row.getCellByColumnName('ID')).pwLocator.textContent();
    if (id === null) throw new Error('ID is null');

    await filterScenario(
      'Filter With ID',
      async () => {
        await tableFilter.filterForm.filter.filterFields.columnName.selectMenuOption('ID');
        await expect(tableFilter.filterForm.filter.filterFields.operator.pwLocator).toHaveText('=');
        await tableFilter.filterForm.filter.filterFields.operator.selectMenuOption('=');
        await tableFilter.filterForm.filter.filterFields.valueNumber.pwLocator.fill(id);
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

        await secondFilterField.valueNumber.pwLocator.fill(id);
      },
      totalExperiments,
    );

    await filterScenario(
      'Show Archived',
      async () => {
        await expect(
          repeatWithFallback(
            async () =>
              await expect(tableFilter.filterForm.showArchived.pwLocator).toHaveAttribute(
                'aria-checked',
                'true',
              ),
            async () => await tableFilter.filterForm.showArchived.pwLocator.click(),
          ),
        ).toPass({ timeout: 30_000 });
      },
      totalExperiments + 1,
    );
  });

  test('Multi-sort menu', async ({ newProject, newWorkspace }) => {
    const multiSortMenu = projectDetailsPage.f_experimentList.tableActionBar.multiSortMenu;
    const secondRow = multiSortMenu.multiSort.rows.nth(1);
    const checkTableOrder = async (firstKey: keyof ExperimentBase) => {
      const experimentList: ExperimentBase[] = JSON.parse(
        await detExecSync(
          `project list-experiments --json ${newWorkspace.response.workspace.name} ${newProject.response.project.name}`,
        ),
      );

      expect(experimentList[0][firstKey] as number).toBeLessThanOrEqual(
        experimentList[experimentList.length - 1][firstKey] as number,
      );
    };

    const sortingScenario = async (
      firstSortBy: string,
      firstSortOrder: string,
      secondSortBy: string,
      secondSortOrder: string,
      scenario: () => Promise<void>,
    ) => {
      await test.step(`Sort by ${firstSortBy} and ${secondSortBy}`, async () => {
        await multiSortMenu.open();
        await multiSortMenu.multiSort.reset.pwLocator.click();
        await multiSortMenu.close();
        await multiSortMenu.open();

        const firstRow = multiSortMenu.multiSort.rows.nth(0);
        await firstRow.column.selectMenuOption(firstSortBy);
        await firstRow.order.selectMenuOption(firstSortOrder);

        await multiSortMenu.multiSort.add.pwLocator.click();

        await secondRow.column.selectMenuOption(secondSortBy);
        await secondRow.order.selectMenuOption(secondSortOrder);

        await multiSortMenu.close();
        await scenario();
        await waitTableStable();
      });
    };

    await sortingScenario('ID', '9 → 0', 'Start time', 'Oldest → Newest', async () => {
      await checkTableOrder('id');
    });

    await sortingScenario('Trial count', '0 → 9', 'Searcher', 'A → Z', async () => {
      await checkTableOrder('numTrials');
    });
  });

  test('Datagrid Functionality Validations', async ({ authedPage }) => {
    const row = projectDetailsPage.f_experimentList.dataGrid.getRowByIndex(0);
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
    const row = projectDetailsPage.f_experimentList.dataGrid.getRowByIndex(0);
    await row.experimentActionDropdown.open();

    // feel free to split actions into their own test cases. this is just a starting point
    await test.step('Edit', async () => {
      const editedValue = safeName('EDITED_EXPERIMENT_NAME');
      await row.experimentActionDropdown.edit.pwLocator.click();
      await row.experimentActionDropdown.editModal.nameInput.pwLocator.fill(editedValue);
      await row.experimentActionDropdown.editModal.footer.submit.pwLocator.click();
      await waitTableStable();
      await expect.soft((await row.getCellByColumnName('Name')).pwLocator).toHaveText(editedValue);
    });
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

  test('DataGrid Action Pause', async () => {
    // datagrid can be slow, perhaps related to [ET-677]
    projectDetailsPage._page.setDefaultTimeout(10000);

    // experiment should initially be paused
    const row = projectDetailsPage.f_experimentList.dataGrid.getRowByIndex(0);
    await expect.soft((await row.getCellByColumnName('State')).pwLocator).toHaveText('paused');

    // resume experiment
    await row.experimentActionDropdown.open();
    await row.experimentActionDropdown.resume.pwLocator.click();
    await expect.soft((await row.getCellByColumnName('State')).pwLocator).not.toHaveText('paused');

    // pause experiment again
    await row.experimentActionDropdown.open();
    await row.experimentActionDropdown.pause.pwLocator.click();
    await expect.soft((await row.getCellByColumnName('State')).pwLocator).toHaveText('paused');
  });

  test('Datagrid Bulk Action', async () => {
    // should probably go last/before move
    await test.step('Kill', async () => {
      type RowType = typeof projectDetailsPage.f_experimentList.dataGrid.rows;
      const rows = [0, 1].map((idx) => {
        return projectDetailsPage.f_experimentList.dataGrid.getRowByIndex(idx);
      });
      const expectStateForRow = async (state: string, row: RowType) => {
        const stateColumn = await row.getCellByColumnName('State');
        await expect(stateColumn.pwLocator).toHaveText(state);
      };
      await rows.reduce(async (memo, row) => {
        await memo;
        await expectStateForRow('paused', row);
        await row.clickColumn('Select');
      }, Promise.resolve());

      await projectDetailsPage.f_experimentList.tableActionBar.actions.kill.select();

      // TODO: modal component model assumes buttons are attached to form
      await projectDetailsPage.pwLocator.getByRole('button', { name: 'kill' }).click();

      await expect(async () => {
        await Promise.all([
          ...rows.map(expectStateForRow.bind(this, 'canceled')),
          expectStateForRow(
            'paused',
            projectDetailsPage.f_experimentList.dataGrid.getRowByIndex(2),
          ),
        ]);
      }).toPass();
    });
  });
});
