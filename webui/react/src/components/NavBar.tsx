import { Dropdown, Menu } from 'antd';
import React from 'react';

import Avatar from 'components/Avatar';
import Logo, { LogoTypes } from 'components/Logo';
import NavItem, { NavItemType } from 'components/NavItem';
import Agents from 'contexts/Agents';
import ClusterOverview from 'contexts/ClusterOverview';

import css from './NavBar.module.scss';

interface Props {
  username?: string;
}

const NavBar: React.FC<Props> = (props: Props) => {
  const overview = ClusterOverview.useStateContext();
  const agents = Agents.useStateContext();

  const menu = (
    <Menu>
      <Menu.Item>
        <NavItem  path={'/det/logout'}>Sign Out</NavItem>
      </Menu.Item>
    </Menu>
  );

  return (
    <nav className={css.base}>
      <Logo type={LogoTypes.Light} />
      <div className={css.group}>
        <NavItem
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
      </div>
    </nav>
  );
};

export default NavBar;
