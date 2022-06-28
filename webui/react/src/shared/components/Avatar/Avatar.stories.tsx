import { text, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import Avatar from './Avatar';

export default {
  component: Avatar,
  decorators: [ withKnobs ],
  title: 'Avatar',
};

export const Default = (): React.ReactNode => <Avatar displayName="Anonymous" />;

export const Custom = (): React.ReactNode => (
  <Avatar displayName={text('Name', 'Martin Luther King')} />);
