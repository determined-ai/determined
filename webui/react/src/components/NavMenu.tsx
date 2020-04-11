import React, { useEffect, useState } from 'react';
import { useLocation } from 'react-router-dom';
import styled, { css } from 'styled-components';
import { switchProp } from 'styled-tools';

import NavItem, { NavItemType } from 'components/NavItem';
import { RouteConfigItem } from 'routes';

export enum NavMenuType {
  Main = 'main',
  SideBar = 'side-bar',
  SideBarIconOnly = 'side-bar-icon-only',
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
  const [ selectedId, setSelectedId ] = useState(props.defaultRouteId);
  const navItemType = menuToItemTypes[props.type || NavMenuType.Main];

  const routes = props.routes;
  const basePath = props.basePath;
  useEffect(() => {
    const matchingPath = routes.find(item => {
      return RegExp(`^${basePath}${item.path}`).test(location.pathname);
    });
    if (matchingPath) setSelectedId(matchingPath.id);
  }, [ location.pathname, basePath, routes ]);

  return (
    <Base {...props}>
      {props.routes.map(route => (
        <NavItem
          active={selectedId === route.id}
          crossover={route.component == null}
          icon={route.icon}
          key={route.id}
          path={route.path}
          popout={route.popout}
          suffixIcon={route.suffixIcon}
          type={navItemType}
        >{props.showLabels && route.title}</NavItem>
      ))}
    </Base>
  );
};

NavMenu.defaultProps = {
  basePath: '',
  showLabels: true,
  type: NavMenuType.Main,
};

const cssMain = css`
  > * { margin-right: 3.2rem; }
  > *:last-child { margin-right: 0; }
`;

const cssSideBar = css`
  flex-direction: column;
  > * { margin-bottom: 1.6rem; }
  > *:last-child { margin-bottom: 0; }
`;

const cssSideBarIconOnly = css`
  flex-direction: column;
  > * { margin-bottom: 1.6rem; }
  > *:last-child { margin-bottom: 0; }
`;

const typeStyles = {
  [NavMenuType.Main]: cssMain,
  [NavMenuType.SideBar]: cssSideBar,
  [NavMenuType.SideBarIconOnly]: cssSideBarIconOnly,
};

const Base = styled.div<Props>`
  display: flex;
  ${switchProp('type', typeStyles)}
`;

export default NavMenu;
