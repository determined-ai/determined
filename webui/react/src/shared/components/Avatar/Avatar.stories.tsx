import { Meta, Story } from '@storybook/react';
import React from 'react';

import useUI from 'shared/contexts/stores/UI';

import Avatar, { Size } from './Avatar';

export default {
  argTypes: {
    darkLight: { table: { disable: true } },
    displayName: { table: { disable: true } },
    nameLength: { control: { max: 3, min: 1, step: 1, type: 'range' } },
    size: { control: { type: 'inline-radio' } },
  },
  component: Avatar,
  title: 'Shared/Avatar',
} as Meta<typeof Avatar>;

type AvatarProps = React.ComponentProps<typeof Avatar>;

const names = ['Admin', 'Determined AI', 'Gold Experience Requiem'];

export const Default: Story<AvatarProps & { nameLength: number }> = ({ nameLength, ...args }) => {
  const { ui } = useUI();
  return <Avatar {...args} darkLight={ui.darkLight} displayName={names[nameLength - 1]} />;
};

Default.args = { nameLength: 1, noColor: false, size: Size.Medium, square: false };
