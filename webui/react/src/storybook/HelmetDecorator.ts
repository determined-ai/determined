import { DecoratorFunction } from '@storybook/addons';
import React from 'react';
import { HelmetProvider } from 'react-helmet-async';

const HelmetDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(HelmetProvider, null, storyFn());
};

export default HelmetDecorator;
