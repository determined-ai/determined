import React, { useState } from 'react';
import { useRouteMatch } from 'react-router-dom';
import styled from 'styled-components';
import { ifProp, theme } from 'styled-tools';

/*
 * Once collapse is supported on Elm, we can uncomment
 * all the commented pieces below to enable it again.
 */

// import NavItem, { NavItemType } from 'components/NavItem';
import NavMenu, { NavMenuType } from 'components/NavMenu';
import { defaultDetRouteId, detRoutes } from 'routes';

interface Props {
  collapsed: boolean;
}

const SideBar: React.FC = () => {
  const { path } = useRouteMatch();
  const [ collapsed ] = useState(false);

  // const handleClick = (): void => setCollapsed(!collapsed);

  return (
    <Base collapsed={collapsed} id="side-menu">
      <NavMenu
        basePath={path}
        defaultRouteId={defaultDetRouteId}
        routes={detRoutes}
        showLabels={!collapsed}
        type={collapsed ? NavMenuType.SideBarIconOnly : NavMenuType.SideBar} />
      <Footer>
        {/* <NavItem icon={collapsed ? 'expand' : 'collapse'}
          type={collapsed ? NavItemType.SideBarIconOnly : NavItemType.SideBar}
          onClick={handleClick}>
          {!collapsed && 'Collapse'}
        </NavItem> */}
        <Version collapsed={collapsed}>
          <span>Version</span>
          <span>{process.env.VERSION}</span>
        </Version>
      </Footer>
    </Base>
  );
};

const Base = styled.div<Props>`
  background-color: #f7f7f7;
  border-right: solid 1px ${theme('colors.monochrome.12')};
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  padding-top: ${theme('sizes.layout.big')};
  width: ${ifProp('collapsed', theme('sizes.sidebar.minWidth'), theme('sizes.sidebar.maxWidth'))};
`;

const Footer = styled.div`
  display: flex;
  flex-direction: column;
`;

const Version = styled.div<Props>`
  background-color: ${theme('colors.monochrome.13')};
  display: flex;
  font-size: ${theme('sizes.font.micro')};
  justify-content: center;
  padding: ${theme('sizes.layout.small')} 0;
  > *:first-child {
    display: ${ifProp('collapsed', 'none', 'flex')};
    padding-right: ${theme('sizes.layout.tiny')};
  }
`;

SideBar.defaultProps = {
  collapsed: true,
};

export default SideBar;
