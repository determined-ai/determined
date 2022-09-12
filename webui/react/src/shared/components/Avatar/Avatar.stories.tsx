import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import { DarkLight } from 'shared/themes';

import Avatar from './Avatar';

export default {
  component: Avatar,
  title: 'Avatar',
} as Meta<typeof Avatar>;

export const Default: ComponentStory<typeof Avatar> = (args) => (
  <Avatar {...args} />
);

Default.args = {
  darkLight: DarkLight.Light,
  displayName: 'Anonymous',
};
