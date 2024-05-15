import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Sort } from 'hew/DataGrid/DataGrid';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { Loaded } from 'hew/utils/loadable';
import { useState } from 'react';

import MultiSortMenu, { ADD_SORT_TEXT, SORT_MENU_TITLE } from './MultiSortMenu';
import { projectColumns } from './MultiSortMenu.test.mock';
import { ThemeProvider } from './ThemeProvider';

const DEFAULT_SORTS: Sort[] = [
  {
    column: 'id',
    direction: 'desc',
  },
];

interface Props {
  onChange: (sorts: Sort[]) => void;
}
const SetupComponent = ({ onChange }: Props) => {
  const [sorts, setSorts] = useState<Sort[]>(DEFAULT_SORTS);
  const handleChange = (sorts: Sort[]) => {
    setSorts(sorts);
    onChange(sorts);
  };
  return <MultiSortMenu columns={Loaded(projectColumns)} sorts={sorts} onChange={handleChange} />;
};
const setup = () => {
  const user = userEvent.setup();
  const handleChange = vi.fn();
  render(
    <UIProvider theme={DefaultTheme.Light}>
      <ThemeProvider>
        <SetupComponent onChange={handleChange} />
      </ThemeProvider>
    </UIProvider>,
  );
  return { handleChange, user };
};

describe('Sort menu', () => {
  it('should display', async () => {
    const { user } = setup();
    await user.click(await screen.findByRole('button'));
    expect(screen.queryByText(SORT_MENU_TITLE)).toBeInTheDocument();
  });
  it('should add sort', async () => {
    const { user, handleChange } = setup();
    await user.click(await screen.findByRole('button'));
    expect(screen.queryByText(SORT_MENU_TITLE)).toBeInTheDocument();
    await user.click(await screen.findByText(ADD_SORT_TEXT));
    expect(handleChange).toHaveBeenCalledWith([
      ...DEFAULT_SORTS,
      {
        column: undefined,
        direction: undefined,
      },
    ]);
    // select new Sort
  });
});
