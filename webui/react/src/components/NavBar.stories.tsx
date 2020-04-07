import React from 'react';

import Agents from 'contexts/Agents';
import RouterDecorator from 'storybook/RouterDecorator';
import { StoryMetadata } from 'storybook/types';

import NavBar from './NavBar';

export default {
  component: NavBar,
  decorators: [ RouterDecorator ],
  title: 'NavBar',
} as StoryMetadata;

export const Default = (): React.ReactNode => (
  <Agents.Provider>
    <NavBar username="determined" />
  </Agents.Provider>
);
