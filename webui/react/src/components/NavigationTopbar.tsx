import { Menu } from 'antd';
import React from 'react';

import Auth from 'contexts/Auth';
import UI from 'contexts/UI';

import Avatar from './Avatar';
import DropdownMenu, { Placement } from './DropdownMenu';
import Link from './Link';
import Logo, { LogoTypes } from './Logo';
import css from './NavigationTopbar.module.scss';

const NavigationTopbar: React.FC = () => {
  const { isAuthenticated, user } = Auth.useStateContext();
  const ui = UI.useStateContext();

  const username = user?.username || 'Anonymous';
  const showNavigation = isAuthenticated && ui.showChrome;

  if (!showNavigation) return null;

  return (
    <nav className={css.base}>
      <Logo type={LogoTypes.OnDarkHorizontal} />
      <div className={css.user}>
        <DropdownMenu
          menu={<Menu>
            <Menu.Item>
              <Link path={'/det/logout'}>Sign Out</Link>
            </Menu.Item>
          </Menu>}
          offset={{ x: 0, y: 8 }}
          placement={Placement.BottomRight}>
          <Avatar hideTooltip name={username} />
        </DropdownMenu>
      </div>
    </nav>
  );
};

export default NavigationTopbar;
