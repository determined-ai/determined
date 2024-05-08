import { fireEvent, render, screen } from '@testing-library/react';
import { DefaultTheme, UIProvider } from 'hew/Theme';
import { Loaded } from 'hew/utils/loadable';

import { V1LocationType } from 'services/api-ts-sdk';
import { ProjectColumn } from 'types';

import ColumnPickerMenu, { COLUMN_PICKER_MENU } from './ColumnPickerMenu';
import { initialVisibleColumns, projectColumns } from './ColumnPickerMenu.test.mock';
import { ThemeProvider } from './ThemeProvider';

const setup = () => {
  const onVisibleColumnChange = vi.fn();
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <ColumnPickerMenu
          initialVisibleColumns={initialVisibleColumns}
          pinnedColumnsCount={0}
          projectColumns={Loaded(projectColumns as ProjectColumn[])}
          projectId={1}
          tabs={[
            V1LocationType.EXPERIMENT,
            [V1LocationType.VALIDATIONS, V1LocationType.TRAINING, V1LocationType.CUSTOMMETRIC],
            V1LocationType.HYPERPARAMETERS,
          ]}
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
  it('should open when button is clicked', async () => {
    setup();
    fireEvent.click(await screen.findByRole('button'));
    expect(screen.getByTestId(COLUMN_PICKER_MENU)).toBeInTheDocument();
  });
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
});
