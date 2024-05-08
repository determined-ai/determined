import { fireEvent, render, screen } from '@testing-library/react';
import { DefaultTheme, UIProvider } from 'hew/Theme';

import { getProjectColumns } from 'services/api';
import { V1LocationType } from 'services/api-ts-sdk';

import ColumnPickerMenu, { COLUMN_PICKER_MENU } from './ColumnPickerMenu';
import { initialVisibleColumns, projectColumns } from './ColumnPickerMenu.test.mock';
import { ThemeProvider } from './ThemeProvider';

vi.mock('services/api', () => ({
  getProjectColumns: vi.fn(),
}));

vi.mock('hew/utils/Loadable');

const setup = () => {
  const fetch = async () => {
    return await getProjectColumns({ id: 1 });
  };
  const columns = fetch();

  const view = render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <ColumnPickerMenu
          initialVisibleColumns={initialVisibleColumns}
          // @ts-expect-error Mock data does not need to be typed
          projectColumns={columns}
          projectId={1}
          tabs={[
            V1LocationType.EXPERIMENT,
            [V1LocationType.VALIDATIONS, V1LocationType.TRAINING, V1LocationType.CUSTOMMETRIC],
            V1LocationType.HYPERPARAMETERS,
          ]}
        />
      </ThemeProvider>
    </UIProvider>,
  );
  return { view };
};

describe('ColumnPickerMenu', () => {
  beforeAll(() => {
    // @ts-expect-error Mock data does not need to be typed
    vi.mocked(getProjectColumns).mockResolvedValue(projectColumns);
  });
  it('should open when button is clicked', async () => {
    setup();
    fireEvent.click(await screen.findByRole('button'));
    expect(screen.getByTestId(COLUMN_PICKER_MENU)).toBeInTheDocument();
  });
});
