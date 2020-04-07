import React, { PropsWithChildren } from 'react';
import styled, { css } from 'styled-components';
import { switchProp, theme } from 'styled-tools';

import Icon from 'components/Icon';
import Label, { LabelTypes } from 'components/Label';
import Link from 'components/Link';

export enum NavItemType {
  Default = 'default',
  Main = 'main',
  SideBar = 'side-bar',
  SideBarIconOnly = 'side-bar-icon-only',
}

interface Props {
  active?: boolean;
  crossover?: boolean;
  icon?: string;
  path?: string;
  popout?: boolean;
  suffixIcon?: string;
  type?: NavItemType;
  onClick?: (event: React.MouseEvent) => void;
}

const itemToLabelTypes = {
  [NavItemType.Default]: undefined,
  [NavItemType.Main]: LabelTypes.NavMain,
  [NavItemType.SideBar]: LabelTypes.NavSideBar,
  [NavItemType.SideBarIconOnly]: undefined,
};

const NavItem: React.FC<Props> = (props: PropsWithChildren<Props>) => {
  const labelType = itemToLabelTypes[props.type || NavItemType.Main];

  const handleClick = (event: React.MouseEvent): void => {
    if (!props.path && props.onClick) props.onClick(event);
  };

  const navItem = (
    <Base {...props} onClick={handleClick}>
      {props.icon && <Icon name={props.icon} />}
      {props.children && <Label type={labelType}>{props.children}</Label>}
      {props.suffixIcon && <Icon name={props.suffixIcon} size="small" />}
    </Base>
  );

  if (props.path) return (
    <Link
      crossover={props.crossover}
      path={props.path}
      popout={props.popout}
      onClick={props.onClick}>{navItem}</Link>
  );

  return navItem;
};

NavItem.defaultProps = {
  active: false,
  crossover: false,
  type: NavItemType.Default,
};

const cssMain = css`
  border-bottom: solid 0.4rem transparent;
  border-top: solid 0.4rem transparent;
  color: #ddd;
  line-height: 4rem;
  &:hover { color: white; }
  &.active {
    border-bottom-color: ${theme('colors.core.primary')};
    color: white;
  }
  & > div:not(:first-child) { padding-left: ${theme('sizes.layout.medium')}; }
  & > i:not(:first-child) { padding-left: ${theme('sizes.layout.tiny')}; }
`;

const cssSideBar = css`
  border-left: solid 0.4rem transparent;
  border-right: solid 0.4rem transparent;
  color: #444;
  font-size: 1.2rem;
  padding: 0.4rem 1.2rem;
  &:hover { color: ${theme('colors.core.action')}; }
  &.active {
    border-left-color: ${theme('colors.core.action')};
    color: ${theme('colors.core.action')};
  }
  & > *:first-child { margin-right: 1.6rem; }
`;

const cssSideBarIconOnly = css`
  border-left: solid 0.4rem transparent;
  border-right: solid 0.4rem transparent;
  color: #444;
  padding: 0.4rem 1rem;
  &:hover { color: ${theme('colors.core.action')}; }
  &.active {
    border-left-color: ${theme('colors.core.action')};
    color: ${theme('colors.core.action')};
  }
  & > *:not(:first-child) { display: none; }
`;

const typeStyles = {
  [NavItemType.Main]: cssMain,
  [NavItemType.SideBar]: cssSideBar,
  [NavItemType.SideBarIconOnly]: cssSideBarIconOnly,
};

const Base = styled.div.attrs((props: Props) => ({
  className: props.active ? 'active' : '',
}))`
  align-items: center;
  color: inherit;
  cursor: pointer;
  display: flex;
  line-height: 1;
  text-decoration: none;
  ${switchProp('type', typeStyles)}
`;

export default NavItem;
