import { DecoratorFunction } from '@storybook/addons';
import React from 'react';
import { BrowserRouter } from 'react-router-dom';

const RouterDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(BrowserRouter, null, storyFn());
};

export default RouterDecorator;
