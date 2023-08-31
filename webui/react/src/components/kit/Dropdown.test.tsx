import { render, screen, waitFor } from '@testing-library/react';
import userEvent, { PointerEventsCheckLevel } from '@testing-library/user-event';
import { PropsWithChildren } from 'react';

import Button from 'components/kit/Button';
import Dropdown, { MenuItem, Props } from 'components/kit/Dropdown';

const MENU_LABEL_1 = 'Menu Option 1';
const MENU_LABEL_2 = 'Menu Option 2';
const MENU_LABEL_3 = 'Menu Option 3';
const MENU: MenuItem[] = [
  { key: 'item1', label: MENU_LABEL_1 },
  { key: 'item2', label: MENU_LABEL_2 },
  { type: 'divider' },
  { key: 'item3', label: MENU_LABEL_3 },
];
const TRIGGER_LABEL = 'Open Dropdown';

const Trigger = () => <Button>{TRIGGER_LABEL}</Button>;

const user = userEvent.setup({ pointerEventsCheck: PointerEventsCheckLevel.Never });

const setup = (props: PropsWithChildren<Props> = { children: Trigger(), menu: MENU }) => {
  const handleClick = vi.fn();
  const view = render(<Dropdown {...props} onClick={handleClick} />);
  return { handleClick, view };
};

describe('Dropdown', () => {
  it('renders dropdown trigger', async () => {
    setup();

    await waitFor(() => {
      expect(screen.queryByRole('button', { name: TRIGGER_LABEL })).toBeInTheDocument();
    });
  });

  it('opens dropdown', async () => {
    setup();

    await user.click(screen.getByRole('button', { name: TRIGGER_LABEL }));

    await waitFor(() => {
      expect(screen.queryByRole('menuitem', { name: MENU_LABEL_1 })).toBeInTheDocument();
      expect(screen.queryByRole('menuitem', { name: MENU_LABEL_2 })).toBeInTheDocument();
      expect(screen.queryByRole('menuitem', { name: MENU_LABEL_3 })).toBeInTheDocument();
    });
  });

  it('select option and closes dropdown', async () => {
    const { handleClick } = setup();

    await user.click(screen.getByRole('button', { name: TRIGGER_LABEL }));
    await waitFor(() => {
      expect(screen.queryByRole('menuitem', { name: MENU_LABEL_1 })).toBeInTheDocument();
    });

    await user.click(screen.getByRole('menuitem', { name: MENU_LABEL_1 }));
    expect(handleClick).toHaveBeenCalled();

    /**
     * The dropdown is appended to the end of the body/document and dismissing the
     * dropdown menu does not remove it, but simply applies a class to hide it.
     */
    await waitFor(() => {
      const menu = document.getElementsByClassName('ant-dropdown');
      expect(menu[0]).toHaveClass('ant-dropdown-hidden');
    });
  });
});
