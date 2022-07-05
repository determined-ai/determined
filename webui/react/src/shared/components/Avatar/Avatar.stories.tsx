import { select, text, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import { DarkLight } from 'shared/themes';

import Avatar from './Avatar';

export default {
  component: Avatar,
  decorators: [ withKnobs ],
  title: 'Avatar',
};

const DARK_LIGHT_OPTIONS = [ DarkLight.Dark, DarkLight.Light ];

export const Default = (): React.ReactNode => (
  <Avatar darkLight={DarkLight.Light} displayName="Anonymous" />
);

export const Custom = (): React.ReactNode => (
  <Avatar
    darkLight={select('Theme', DARK_LIGHT_OPTIONS, DarkLight.Light)}
    displayName={text('Name', 'Martin Luther King')}
  />
);
