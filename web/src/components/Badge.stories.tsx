import { select, text, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import { enumToOptions } from 'storybook/utils';
import { CommandState } from 'types';

import Badge, { BadgeType } from './Badge';

export default {
  component: Badge,
  decorators: [ withKnobs ],
  title: 'Badge',
};

const knobTypeOptions = enumToOptions<BadgeType>(BadgeType);

export const Default = (): React.ReactNode => <Badge>a4fdb98</Badge>;

export const Custom = (): React.ReactNode => (
  <Badge
    state={select('State', CommandState, CommandState.Assigned)}
    type={select<BadgeType>('Type', knobTypeOptions, BadgeType.Default)}>
    {text('Content', 'a4fdb98')}
  </Badge>
);
