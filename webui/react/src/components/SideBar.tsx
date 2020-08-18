import { Tooltip } from 'antd';
import React, { useCallback, useState } from 'react';

/*
 * Once collapse is supported on Elm, we can uncomment
 * all the commented pieces below to enable it again.
 */

import { sidebarRoutes } from 'routes';

import NavItem, { NavItemType } from './NavItem';
import NavMenu, { NavMenuType } from './NavMenu';
import css from './SideBar.module.scss';

interface Props {
  collapsed?: boolean;
}

const SideBar: React.FC<Props> = (props: Props) => {
  const [ collapsed, setCollapsed ] = useState(props.collapsed);
  const classes = [ css.base ];
  const shortVersion = (process.env.VERSION || '').split('.').slice(0, 3).join('.');

  if (collapsed) classes.push(css.collapsed);

  const handleClick = useCallback((): void => {
    setCollapsed(prevCollapsed => !prevCollapsed);
  }, [ setCollapsed ]);

  return (
    <div className={classes.join(' ')} id="side-menu">
      <NavMenu
        routes={sidebarRoutes}
        showLabels={!collapsed}
        type={collapsed ? NavMenuType.SideBarIconOnly : NavMenuType.SideBar} />
      <div className={css.footer}>
        <NavItem
          icon="logs"
          path="/det/logs"
          popout={true}
          type={collapsed ? NavItemType.SideBarIconOnly : NavItemType.SideBar}>
          Master Logs
        </NavItem>
        {process.env.IS_DEV && <NavItem icon={collapsed ? 'expand' : 'collapse'}
          type={collapsed ? NavItemType.SideBarIconOnly : NavItemType.SideBar}
          onClick={handleClick}>
          {!collapsed && 'Collapse'}
        </NavItem>}
        <div className={css.version}>
          <span>Version</span>
          {!collapsed
            ? <span>{process.env.VERSION}</span>
            : <Tooltip placement="bottomLeft" title={`Version ${process.env.VERSION}`}>
              <span>{shortVersion}</span>
            </Tooltip>}
        </div>
      </div>
    </div>
  );
};

SideBar.defaultProps = {
  collapsed: false,
};

export default SideBar;
