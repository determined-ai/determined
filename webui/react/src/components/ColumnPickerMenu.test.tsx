import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DefaultTheme, UIProvider } from 'hew/Theme';
import { Loaded } from 'hew/utils/loadable';
import _ from 'lodash';

import { V1LocationType } from 'services/api-ts-sdk';

import ColumnPickerMenu, { COLUMNS_MENU_BUTTON, LOCATION_LABEL_MAP } from './ColumnPickerMenu';
import { initialVisibleColumns, projectColumns } from './ColumnPickerMenu.test.mock';
import { ThemeProvider } from './ThemeProvider';

const locations = [
  V1LocationType.EXPERIMENT,
  [V1LocationType.VALIDATIONS, V1LocationType.TRAINING, V1LocationType.CUSTOMMETRIC],
  V1LocationType.HYPERPARAMETERS,
  V1LocationType.RUNMETADATA,
];

const PINNED_COLUMNS_COUNT = 0;

const setup = (initCols?: string[]) => {
  const user = userEvent.setup();
  const onVisibleColumnChange = vi.fn();
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <ColumnPickerMenu
          defaultVisibleColumns={initialVisibleColumns}
          initialVisibleColumns={initCols ?? initialVisibleColumns}
          pinnedColumnsCount={PINNED_COLUMNS_COUNT}
          projectColumns={Loaded(projectColumns)}
          projectId={1}
          tabs={locations}
          onVisibleColumnChange={onVisibleColumnChange}
        />
      </ThemeProvider>
    </UIProvider>,
  );
  return {
    onVisibleColumnChange,
    user,
  };
};

describe('ColumnPickerMenu', () => {
  it('should deselect columns', async () => {
    const { onVisibleColumnChange, user } = setup();
    await user.click(await screen.findByTestId(COLUMNS_MENU_BUTTON));
    const columnId = initialVisibleColumns[0];
    const displayName = projectColumns.find((c) => c.column === columnId)?.displayName;
    await user.click(await screen.findByText(displayName ?? ''));
    expect(onVisibleColumnChange).toHaveBeenCalledWith(
      initialVisibleColumns.filter((c) => c !== columnId),
      PINNED_COLUMNS_COUNT,
    );
  });

  it('should select columns', async () => {
    const { onVisibleColumnChange, user } = setup();
    await user.click(await screen.findByTestId(COLUMNS_MENU_BUTTON));
    const unselectedInitialColumns = projectColumns.filter(
      (c) => !initialVisibleColumns.includes(c.column),
    );
    const columnId = unselectedInitialColumns.map((c) => c.column)[0];
    const displayName = projectColumns.find((c) => c.column === columnId)?.displayName;
    await user.click(await screen.findByText(displayName ?? ''));
    const expectedColumns = [...initialVisibleColumns, columnId];
    expect(onVisibleColumnChange).toHaveBeenCalledWith(expectedColumns, PINNED_COLUMNS_COUNT);
  });

  it('should reset', async () => {
    const { onVisibleColumnChange, user } = setup();
    await user.click(await screen.findByTestId(COLUMNS_MENU_BUTTON));
    const columnId = initialVisibleColumns[0];
    const displayName = projectColumns.find((c) => c.column === columnId)?.displayName;
    await user.click(await screen.findByText(displayName ?? ''));
    expect(onVisibleColumnChange).toHaveBeenCalledWith(
      initialVisibleColumns.filter((c) => c !== columnId),
      PINNED_COLUMNS_COUNT,
    );
    const resets = await screen.findAllByText('Reset');
    await user.click(resets[0]);
    expect(onVisibleColumnChange).toHaveBeenCalledWith(initialVisibleColumns);
  });

  it('should switch tabs and display correct columns', async () => {
    const { user } = setup();
    await user.click(await screen.findByTestId(COLUMNS_MENU_BUTTON));
    const tabs = new Set<string>(Object.values(LOCATION_LABEL_MAP));
    const testTab = async (tabName: string) => {
      await user.click(await screen.findByText(tabName));
      const locationForTab = _.findKey(LOCATION_LABEL_MAP, (v) => v === tabName);
      const columnsForLocation = projectColumns.filter((c) => c.location === locationForTab);
      const column = columnsForLocation[0];
      const displayName = column.displayName?.length ? column.displayName : column.column;
      expect(await screen.findByText(displayName)).toBeInTheDocument();
    };

    const availableTabs = Array.from(tabs).filter((tab) => tab !== 'Unspecified');
    for (const tab of availableTabs) {
      await testTab(tab);
    }
  });

  it('should show all', async () => {
    const { onVisibleColumnChange, user } = setup();
    await user.click(await screen.findByTestId(COLUMNS_MENU_BUTTON));
    const showAlls = await screen.findAllByText('Show all');
    await user.click(showAlls[0]);
    const locationColumns = projectColumns
      .filter((c) => c.location === locations[0])
      .map((c) => c.column);
    const addedColumns = _.difference(locationColumns, initialVisibleColumns);
    expect(onVisibleColumnChange).toHaveBeenCalledWith(
      [...initialVisibleColumns, ...addedColumns],
      PINNED_COLUMNS_COUNT,
    );
  });

  it('should hide all', async () => {
    const locationColumns = projectColumns
      .filter((c) => c.location === locations[0])
      .map((c) => c.column);
    const { onVisibleColumnChange, user } = setup(locationColumns);
    await user.click(await screen.findByTestId(COLUMNS_MENU_BUTTON));
    const hideAlls = await screen.findAllByText('Hide all');
    await user.click(hideAlls[0]);
    expect(onVisibleColumnChange).toHaveBeenCalledWith([], PINNED_COLUMNS_COUNT);
  });

  it('should filter', async () => {
    const { user } = setup();
    await user.click(await screen.findByTestId(COLUMNS_MENU_BUTTON));
    const includedColumn = projectColumns.find((c) => c.column === initialVisibleColumns[0]);
    const excludedColumn = projectColumns.find((c) => c.column === initialVisibleColumns[1]);
    await user.type(await screen.findByRole('textbox'), includedColumn?.displayName ?? '');
    expect(screen.queryByText(includedColumn?.displayName ?? '')).toBeInTheDocument();
    expect(screen.queryByText(excludedColumn?.displayName ?? '')).not.toBeInTheDocument();
  });
});
