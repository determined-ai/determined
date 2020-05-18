import React, { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';

import NavItem, { NavItemType } from 'components/NavItem';
import { RouteConfigItem } from 'routes';

import css from './NavMenu.module.scss';

export enum NavMenuType {
  Main = 'main',
  SideBar = 'sidebar',
  SideBarIconOnly = 'sidebarIconOnly',
}

interface Props {
  basePath?: string;
  defaultRouteId: string;
  routes: RouteConfigItem[];
  showLabels?: boolean;
  type?: NavMenuType;
}

const menuToItemTypes = {
  [NavMenuType.Main]: NavItemType.Main,
  [NavMenuType.SideBar]: NavItemType.SideBar,
  [NavMenuType.SideBarIconOnly]: NavItemType.SideBarIconOnly,
};

const NavMenu: React.FC<Props> = (props: Props) => {
  const location = useLocation();
  const { basePath, routes } = props;
  const [ selectedId, setSelectedId ] = useState(props.defaultRouteId);
  const navMenuType = props.type || NavMenuType.Main;
  const navItemType = menuToItemTypes[navMenuType];
  const classes = [ css.base, css[navMenuType] ];

  useEffect(() => {
    const matchingPath = routes.find(item => {
      return RegExp(`^${basePath}${item.path}`).test(location.pathname);
    });
    if (matchingPath) setSelectedId(matchingPath.id);
  }, [ basePath, location.pathname, routes ]);

  return (
    <div className={classes.join(' ')}>
      {props.routes.map(route => (
        <NavItem
          active={selectedId === route.id}
          icon={route.icon}
          key={route.id}
          path={route.path}
          popout={route.popout}
          suffixIcon={route.suffixIcon}
          type={navItemType}
        >{props.showLabels && route.title}</NavItem>
      ))}
    </div>
  );
};

NavMenu.defaultProps = {
  basePath: '',
  showLabels: true,
  type: NavMenuType.Main,
};

export default NavMenu;
