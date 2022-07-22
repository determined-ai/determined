import { select, text, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import { enumToOptions } from 'storybook/utils';
import { CommandState } from 'types';

import { BadgeType } from './Badge';
import BadgeTag from './BadgeTag';

export default {
  component: BadgeTag,
  decorators: [ withKnobs ],
  title: 'BadgeTag',
};

const knobTypeOptions = enumToOptions<BadgeType>(BadgeType);

export const Default = (): React.ReactNode => <BadgeTag label="Special ID">a4fdb98</BadgeTag>;

export const Custom = (): React.ReactNode => (
  <BadgeTag
    label={text('Label', 'Label')}
    preLabel={text('Pre Label', '')}
    state={select('State', CommandState, CommandState.Assigned)}
    type={select<BadgeType>('Type', knobTypeOptions, BadgeType.Default)}>
    {text('Content', 'a4fdb98')}
  </BadgeTag>
);
