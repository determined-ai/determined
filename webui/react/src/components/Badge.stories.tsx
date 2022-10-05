import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import { CommandState } from 'types';

import Badge, { BadgeType } from './Badge';

export default {
  argTypes: {
    children: { name: 'text' },
    state: { control: 'inline-radio', options: CommandState },
    type: { control: 'inline-radio', options: BadgeType },
  },
  component: Badge,
  title: 'Determined/Badges/Badge',
} as Meta<typeof Badge>;

export const Default: ComponentStory<typeof Badge> = (args) => <Badge {...args} />;

Default.args = { children: 'a4fdb98', state: CommandState.Running, type: BadgeType.Default };
