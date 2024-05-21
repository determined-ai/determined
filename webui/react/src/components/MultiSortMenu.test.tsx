import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Sort } from 'hew/DataGrid/DataGrid';
import UIProvider, { DefaultTheme } from 'hew/Theme';
import { Loaded } from 'hew/utils/loadable';
import { useState } from 'react';

import MultiSortMenu, {
  ADD_SORT_TEXT,
  EMPTY_SORT,
  REMOVE_SORT_TITLE,
  RESET_SORT_TEXT,
  SORT_MENU_BUTTON,
  SORT_MENU_TITLE,
} from './MultiSortMenu';
import { projectColumns } from './MultiSortMenu.test.mock';
import { ThemeProvider } from './ThemeProvider';

const initialColumn = projectColumns[0];
const addedColumn = projectColumns[1];
const defaultDirection = 'asc';

const INITIAL_SORT: Sort = {
  column: initialColumn.column,
  direction: defaultDirection,
};

const ADDED_SORT: Sort = {
  column: addedColumn.column,
  direction: defaultDirection,
};

interface Props {
  onChange: (sorts: Sort[]) => void;
}
const SetupComponent = ({ onChange }: Props) => {
  const [sorts, setSorts] = useState<Sort[]>([INITIAL_SORT]);
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
    await user.click(await screen.findByTestId(SORT_MENU_BUTTON));
    expect(screen.queryByText(SORT_MENU_TITLE)).toBeInTheDocument();
  });
  it('should add sort', async () => {
    const { user, handleChange } = setup();
    await user.click(await screen.findByTestId(SORT_MENU_BUTTON));
    expect(screen.queryByText(SORT_MENU_TITLE)).toBeInTheDocument();
    await user.click(await screen.findByText(ADD_SORT_TEXT));
    expect(handleChange).toHaveBeenNthCalledWith(1, [INITIAL_SORT, EMPTY_SORT]);
    const comboboxes = await screen.findAllByRole('combobox');
    await user.click(comboboxes[2]);
    const option = await screen.findByText(addedColumn.displayName ?? '');
    await user.click(option);
    expect(handleChange).toHaveBeenNthCalledWith(2, [INITIAL_SORT, ADDED_SORT]);
  });
  it('should remove sort', async () => {
    const { user, handleChange } = setup();
    await user.click(await screen.findByTestId(SORT_MENU_BUTTON));
    expect(screen.queryByText(SORT_MENU_TITLE)).toBeInTheDocument();
    await user.click(await screen.findByLabelText(REMOVE_SORT_TITLE));
    expect(handleChange).toHaveBeenCalledWith([EMPTY_SORT]);
  });
  it('should reset', async () => {
    const { user, handleChange } = setup();
    await user.click(await screen.findByTestId(SORT_MENU_BUTTON));
    expect(screen.queryByText(SORT_MENU_TITLE)).toBeInTheDocument();
    await user.click(await screen.findByText(RESET_SORT_TEXT));
    expect(handleChange).toHaveBeenCalledWith([EMPTY_SORT]);
  });
});
