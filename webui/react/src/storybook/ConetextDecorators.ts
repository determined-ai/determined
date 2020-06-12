import { DecoratorFunction } from '@storybook/addons';
import React from 'react';

import Experiments from 'contexts/Experiments';

export const ExperimentsDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(Experiments.Provider, null, storyFn());
};
