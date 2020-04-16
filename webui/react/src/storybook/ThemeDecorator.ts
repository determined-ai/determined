import { DecoratorFunction } from '@storybook/addons';
import React from 'react';

import useTheme from 'hooks/useTheme';

const ThemeDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  useTheme();
  return React.createElement('div', null, storyFn());
};

export default ThemeDecorator;
