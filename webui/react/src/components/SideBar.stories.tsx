import React from 'react';

import RouterDecorator from 'storybook/RouterDecorator';
import { StoryMetadata } from 'storybook/types';

import SideBar from './SideBar';

export default {
  component: SideBar,
  decorators: [ RouterDecorator ],
  title: 'SideBar',
} as StoryMetadata;

export const Default = (): React.ReactNode => <SideBar />;
