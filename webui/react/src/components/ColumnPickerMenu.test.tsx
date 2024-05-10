import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { DefaultTheme, UIProvider } from 'hew/Theme';
import { Loaded } from 'hew/utils/loadable';
import _ from 'lodash';

import { V1LocationType } from 'services/api-ts-sdk';
import { ProjectColumn } from 'types';

import ColumnPickerMenu, { locationLabelMap } from './ColumnPickerMenu';
import { initialVisibleColumns, projectColumns } from './ColumnPickerMenu.test.mock';
import { ThemeProvider } from './ThemeProvider';

const locations = [
  V1LocationType.EXPERIMENT,
  [V1LocationType.VALIDATIONS, V1LocationType.TRAINING, V1LocationType.CUSTOMMETRIC],
  V1LocationType.HYPERPARAMETERS,
];

const setup = (initCols?: string[]) => {
  const onVisibleColumnChange = vi.fn();
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <ColumnPickerMenu
          initialVisibleColumns={initCols ?? initialVisibleColumns}
          pinnedColumnsCount={0}
          projectColumns={Loaded(projectColumns as ProjectColumn[])}
          projectId={1}
          tabs={locations}
          onVisibleColumnChange={onVisibleColumnChange}
        />
      </ThemeProvider>
    </UIProvider>,
  );
  return {
    onVisibleColumnChange,
  };
};

describe('ColumnPickerMenu', () => {
  it('should deselect columns', async () => {
    const { onVisibleColumnChange } = setup();
    fireEvent.click(await screen.findByRole('button'));
    const columnId = initialVisibleColumns[0];
    const displayName = projectColumns.find((c) => c.column === columnId)?.displayName;
    fireEvent.click(await screen.findByText(displayName ?? ''));
    expect(onVisibleColumnChange).toHaveBeenCalledWith(
      initialVisibleColumns.filter((c) => c !== columnId),
    );
  });
  it('should select columns', async () => {
    const { onVisibleColumnChange } = setup();
    fireEvent.click(await screen.findByRole('button'));
    const unselectedInitialColumns = projectColumns.filter(
      (c) => !initialVisibleColumns.includes(c.column),
    );
    const columnId = unselectedInitialColumns.map((c) => c.column)[0];
    const displayName = projectColumns.find((c) => c.column === columnId)?.displayName;
    fireEvent.click(await screen.findByText(displayName ?? ''));
    const expectedColumns = [...initialVisibleColumns, columnId];
    expect(onVisibleColumnChange).toHaveBeenCalledWith(expectedColumns);
  });
  it('should reset', async () => {
    const { onVisibleColumnChange } = setup();
    fireEvent.click(await screen.findByRole('button'));
    const columnId = initialVisibleColumns[0];
    const displayName = projectColumns.find((c) => c.column === columnId)?.displayName;
    fireEvent.click(await screen.findByText(displayName ?? ''));
    expect(onVisibleColumnChange).toHaveBeenCalledWith(
      initialVisibleColumns.filter((c) => c !== columnId),
    );
    const resets = await screen.findAllByText('Reset');
    fireEvent.click(resets[0]);
    expect(onVisibleColumnChange).toHaveBeenCalledWith(initialVisibleColumns);
  });
  it('should switch tabs and display correct columns', async () => {
    setup();
    fireEvent.click(await screen.findByRole('button'));
    const tabs = new Set<string>(Object.values(locationLabelMap));
    const testTab = async (tabName: string) => {
      fireEvent.click(await screen.findByText(tabName));
      const locationForTab = _.findKey(locationLabelMap, (v) => v === tabName);
      const columnsForLocation = projectColumns.filter((c) => c.location === locationForTab);
      const column = columnsForLocation[0];
      expect(await screen.findByText(column.displayName)).toBeInTheDocument();
    };
    Array.from(tabs).forEach((tabName: string) => {
      testTab(tabName);
    });
  });
  it('should show all', async () => {
    const { onVisibleColumnChange } = setup();
    fireEvent.click(await screen.findByRole('button'));
    const showAlls = await screen.findAllByText('Show all');
    fireEvent.click(showAlls[0]);
    const locationColumns = projectColumns
      .filter((c) => c.location === locations[0])
      .map((c) => c.column);
    const addedColumns = _.difference(locationColumns, initialVisibleColumns);
    expect(onVisibleColumnChange).toHaveBeenCalledWith([...initialVisibleColumns, ...addedColumns]);
  });
  it('should hide all', async () => {
    const locationColumns = projectColumns
      .filter((c) => c.location === locations[0])
      .map((c) => c.column);
    const { onVisibleColumnChange } = setup(locationColumns);
    fireEvent.click(await screen.findByRole('button'));
    const hideAlls = await screen.findAllByText('Hide all');
    fireEvent.click(hideAlls[0]);
    expect(onVisibleColumnChange).toHaveBeenCalledWith([]);
  });
  it('should filter', async () => {
    setup();
    fireEvent.click(await screen.findByRole('button'));
    const includedColumn = projectColumns.find((c) => c.column === initialVisibleColumns[0]);
    const excludedColumn = projectColumns.find((c) => c.column === initialVisibleColumns[1]);
    fireEvent.change(await screen.findByRole('textbox'), includedColumn?.displayName);
    waitFor(async () => {
      expect(await screen.findAllByText(includedColumn?.displayName ?? '')).toHaveLength(1);
      expect(await screen.findAllByText(excludedColumn?.displayName ?? '')).toHaveLength(0);
    });
  });
});
