import { Dropdown, Menu } from 'antd';
import React from 'react';
import styled from 'styled-components';
import { theme } from 'styled-tools';

import Avatar from 'components/Avatar';
import Logo from 'components/Logo';
import NavItem, { NavItemType } from 'components/NavItem';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';

interface Props {
  username?: string;
}

const NavBar: React.FC<Props> = (props: Props) => {
  const overview = ClusterOverview.useStateContext();
  const agents = Agents.useStateContext();

  const menu = (
    <Menu>
      <Menu.Item>
        <NavItem crossover={true} path={'/ui/logout'}>Sign Out</NavItem>
      </Menu.Item>
    </Menu>
  );

  return (
    <Base>
      <Logo />
      <Group>
        <NavItem
          crossover={true}
          icon="cluster"
          path="/ui/cluster"
          type={NavItemType.Main}>
          {agents.hasLoaded &&
            (overview.totalResources.total !== 0 ? `${overview.allocation}%` : 'No Agents')}
        </NavItem>
        <NavItem
          icon=""
          path="/docs"
          popout={true}
          suffixIcon="popout"
          type={NavItemType.Main}>
          Docs
        </NavItem>
        <Dropdown overlay={menu} trigger={[ 'click' ]}>
          <a className="ant-dropdown-link" href="#">
            <Avatar name={props.username || 'Anonymous'} />
          </a>
        </Dropdown>
      </Group>
    </Base>
  );
};

const Base = styled.nav`
  align-items: center;
  background-color: ${theme('colors.core.secondary')};
  display: flex;
  flex-shrink: 0;
  height: ${theme('sizes.navbar.height')};
  justify-content: space-between;
  padding: 0 1.6rem;
`;

const Group = styled.div`
  align-items: center;
  display: flex;
  > *:not(:first-child) { margin-left: ${theme('sizes.layout.huge')}; }
`;

export default NavBar;
