import { DecoratorFunction } from '@storybook/addons';
import React from 'react';
import { ThemeProvider } from 'styled-components';

import { lightTheme } from 'themes';

const ThemeDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(ThemeProvider, { theme: lightTheme }, storyFn());
};

export default ThemeDecorator;
