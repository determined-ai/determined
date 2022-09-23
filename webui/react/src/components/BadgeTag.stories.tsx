import { ComponentStory, Meta } from '@storybook/react';
import React from 'react';

import { CommandState } from 'types';

import { BadgeType } from './Badge';
import BadgeTag from './BadgeTag';

export default {
  argTypes: {
    children: { name: 'text' },
    label: { control: 'text' },
    prelabel: { control: 'text' },
    state: { control: 'inline-radio', options: CommandState },
    type: { control: 'inline-radio', options: BadgeType },
  },
  component: BadgeTag,
  title: 'Determined/Badges/BadgeTag',
} as Meta<typeof BadgeTag>;

export const Default: ComponentStory<typeof BadgeTag> = (args) => <BadgeTag {...args} />;

Default.args = {
  children: 'a4fdb98',
  label: 'Label',
  state: CommandState.Running,
  type: BadgeType.Default,
};
