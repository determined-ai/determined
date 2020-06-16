import React, { PropsWithChildren, useCallback } from 'react';

import Icon from 'components/Icon';
import Label, { LabelTypes } from 'components/Label';
import Link from 'components/Link';

import css from './NavItem.module.scss';

export enum NavItemType {
  Default = 'default',
  Main = 'main',
  SideBar = 'sidebar',
  SideBarIconOnly = 'sidebarIconOnly',
}

interface Props {
  active?: boolean;
  icon?: string;
  path?: string;
  popout?: boolean;
  suffixIcon?: string;
  type?: NavItemType;
  title?: string;
  onClick?: (event: React.MouseEvent) => void;
}

const itemToLabelTypes = {
  [NavItemType.Default]: undefined,
  [NavItemType.Main]: LabelTypes.NavMain,
  [NavItemType.SideBar]: LabelTypes.NavSideBar,
  [NavItemType.SideBarIconOnly]: undefined,
};

const NavItem: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const navItemType = props.type || NavItemType.Main;
  const labelType = itemToLabelTypes[navItemType];
  const classes = [ css.base, css[navItemType] ];

  if (props.active) classes.push(css.active);

  const handleClick = useCallback((event: React.MouseEvent): void => {
    if (!props.path && props.onClick) props.onClick(event);
  }, [ props ]);

  const navItem = (
    <div className={classes.join(' ')} title={props.title} onClick={handleClick}>
      {props.icon && <Icon name={props.icon} />}
      {props.children && <Label type={labelType}>{props.children}</Label>}
      {props.suffixIcon && <Icon name={props.suffixIcon} size="small" />}
    </div>
  );

  if (props.path) return (
    <Link
      inherit
      path={props.path}
      popout={props.popout}
      onClick={props.onClick}>{navItem}</Link>
  );

  return navItem;
};

NavItem.defaultProps = {
  active: false,
  type: NavItemType.Default,
};

export default NavItem;
