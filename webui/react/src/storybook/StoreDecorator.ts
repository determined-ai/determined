import { DecoratorFunction } from '@storybook/addons';
import React from 'react';

import StoreProvider from 'contexts/Store';

const StoreDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(StoreProvider, null, storyFn());
};

export default StoreDecorator;
