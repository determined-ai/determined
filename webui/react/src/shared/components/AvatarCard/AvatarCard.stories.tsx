import { Meta, Story } from '@storybook/react';
import React from 'react';

import useUI from 'shared/contexts/stores/UI';

import { Size } from '../Avatar/Avatar';

import AvatarCard from './AvatarCard';

export default {
  argTypes: {
    darkLight: { table: { disable: true } },
    displayName: { table: { disable: true } },
    nameLength: { control: { max: 3, min: 1, step: 1, type: 'range' } },
    size: { control: { type: 'inline-radio' } },
  },
  component: AvatarCard,
  title: 'Shared/Avatar Card',
} as Meta<typeof AvatarCard>;

type AvatarCardProps = React.ComponentProps<typeof AvatarCard>;

const names = ['Admin', 'Determined AI', 'Gold Experience Requiem'];

export const Default: Story<AvatarCardProps & { nameLength: number }> = ({
  nameLength,
  ...args
}) => {
  const { ui } = useUI();
  return <AvatarCard {...args} darkLight={ui.darkLight} displayName={names[nameLength - 1]} />;
};

Default.args = { nameLength: 1, noColor: false, size: Size.Medium, square: false };
