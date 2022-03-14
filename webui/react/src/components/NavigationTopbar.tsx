import { Menu } from 'antd';
import React from 'react';

import { useStore } from 'contexts/Store';
import { paths } from 'routes/utils';

import Avatar from './Avatar';
import Dropdown, { Placement } from './Dropdown';
import Link from './Link';
import Logo, { LogoType } from './Logo';
import css from './NavigationTopbar.module.scss';

const NavigationTopbar: React.FC = () => {
  const { auth, info, ui } = useStore();

  const showNavigation = auth.isAuthenticated && ui.showChrome;

  if (!showNavigation) return null;

  return (
    <nav className={css.base}>
      <Logo branding={info.branding} type={LogoType.OnDarkHorizontal} />
      <div className={css.user}>
        <Dropdown
          content={(
            <Menu>
              <Menu.Item key="sign-out">
                <Link path={paths.logout()}>Sign Out</Link>
              </Menu.Item>
            </Menu>
          )}
          offset={{ x: 0, y: 8 }}
          placement={Placement.BottomRight}>
          <Avatar hideTooltip username={auth.user?.username} />
        </Dropdown>
      </div>
    </nav>
  );
};

export default NavigationTopbar;
