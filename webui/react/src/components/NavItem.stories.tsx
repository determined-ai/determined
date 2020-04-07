import React from 'react';

import RouterDecorator from 'storybook/RouterDecorator';
import { StoryMetadata } from 'storybook/types';
import { lightTheme } from 'themes';

import NavItem, { NavItemType } from './NavItem';

export default {
  component: NavItem,
  decorators: [ RouterDecorator ],
  title: 'NavItem',
} as StoryMetadata;

const mainStory = {
  parameters: {
    backgrounds: [
      { default: true, name: 'main background', value: lightTheme.colors.core.secondary },
    ],
  },
};

const sideBarStory = {
  parameters: {
    backgrounds: [
      { default: true, name: 'sidebar background', value: lightTheme.colors.monochrome[0] },
    ],
  },
};

export const Default = (): React.ReactNode => <NavItem>Plain Route</NavItem>;

export const Main = (): React.ReactNode => <NavItem type={NavItemType.Main}>Home</NavItem>;

Main.story = mainStory;

export const MainActive = (): React.ReactNode => (
  <NavItem active={true} type={NavItemType.Main}>Home</NavItem>
);

MainActive.story = mainStory;

export const SideBar = (): React.ReactNode => (
  <NavItem type={NavItemType.SideBar}>Dashboard</NavItem>
);

SideBar.story = sideBarStory;

export const SideBarActive = (): React.ReactNode => (
  <NavItem active={true} type={NavItemType.SideBar}>Experiment</NavItem>
);

SideBarActive.story = sideBarStory;

export const SideBarWithIcon = (): React.ReactNode => (
  <NavItem icon="star" type={NavItemType.SideBar}>Dashboard</NavItem>
);

SideBarWithIcon.story = sideBarStory;

export const SideBarActiveWithIcon = (): React.ReactNode => (
  <NavItem active={true} icon="star" type={NavItemType.SideBar}>Dashboard</NavItem>
);

SideBarActiveWithIcon.story = sideBarStory;

export const SideBarIconOnly = (): React.ReactNode => (
  <NavItem icon="notebook" type={NavItemType.SideBarIconOnly} />
);

SideBarIconOnly.story = sideBarStory;

export const SideBarActiveIconOnly = (): React.ReactNode => (
  <NavItem active={true} icon="notebook" type={NavItemType.SideBarIconOnly} />
);

SideBarActiveIconOnly.story = sideBarStory;
