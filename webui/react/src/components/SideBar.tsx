import React, { useState } from 'react';
import { useRouteMatch } from 'react-router-dom';

/*
 * Once collapse is supported on Elm, we can uncomment
 * all the commented pieces below to enable it again.
 */

// import NavItem, { NavItemType } from 'components/NavItem';
import NavMenu, { NavMenuType } from 'components/NavMenu';
import { defaultSideBarRoute, sidebarRoutes } from 'routes';

import css from './SideBar.module.scss';

interface Props {
  collapsed?: boolean;
}

const SideBar: React.FC<Props> = (props: Props) => {
  const { path } = useRouteMatch();
  const [ collapsed ] = useState(props.collapsed);
  const classes = [ css.base ];

  if (collapsed) classes.push(css.collapsed);

  // const handleClick = useCallback((): void => setCollapsed(!collapsed), [ setCollapsed ]);

  return (
    <div className={classes.join(' ')} id="side-menu">
      <NavMenu
        basePath={path}
        defaultRouteId={defaultSideBarRoute.id}
        routes={sidebarRoutes}
        showLabels={!collapsed}
        type={collapsed ? NavMenuType.SideBarIconOnly : NavMenuType.SideBar} />
      <div className={css.footer}>
        {/* <NavItem icon={collapsed ? 'expand' : 'collapse'}
          type={collapsed ? NavItemType.SideBarIconOnly : NavItemType.SideBar}
          onClick={handleClick}>
          {!collapsed && 'Collapse'}
        </NavItem> */}
        <div className={css.version}>
          <span>Version</span>
          <span>{process.env.VERSION}</span>
        </div>
      </div>
    </div>
  );
};

SideBar.defaultProps = {
  collapsed: false,
};

export default SideBar;
