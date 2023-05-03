import { Dropdown as AntDropdown } from 'antd';
import { MenuProps } from 'antd/es/menu/menu';
import { PropsWithChildren, useMemo } from 'react';
import * as React from 'react';

interface MenuDivider {
  type: 'divider';
}

interface MenuOption {
  className?: string;
  danger?: boolean;
  disabled?: boolean;
  key: string;
  label: React.ReactNode;
  icon?: React.ReactNode;
  type?: 'item';
}

export type MenuItem = MenuDivider | MenuOption;

export type Placement =
  | 'topLeft'
  | 'topCenter'
  | 'topRight'
  | 'bottomLeft'
  | 'bottomCenter'
  | 'bottomRight'
  | 'top'
  | 'bottom';

export type DropdownEvent = React.MouseEvent<HTMLElement> | React.KeyboardEvent<HTMLElement>;

interface Props {
  disabled?: boolean;
  isContextMenu?: boolean;
  menu: MenuItem[];
  open?: boolean;
  placement?: Placement;
  onClick?:
    | ((key: string, e: DropdownEvent) => void)
    | ((key: string, e: DropdownEvent) => Promise<void>);
}
const Dropdown: React.FC<PropsWithChildren<Props>> = ({
  children,
  disabled,
  isContextMenu,
  menu,
  open,
  placement = 'bottomLeft',
  onClick,
}) => {
  const antdMenu: MenuProps = useMemo(() => {
    return {
      items: menu,
      onClick: (info) => {
        info.domEvent.stopPropagation();
        onClick?.(info.key, info.domEvent);
      },
    };
  }, [menu, onClick]);

  return (
    <AntDropdown
      disabled={disabled}
      menu={antdMenu}
      open={open}
      placement={placement}
      trigger={[isContextMenu ? 'contextMenu' : 'click']}>
      {children}
    </AntDropdown>
  );
};

export default Dropdown;
