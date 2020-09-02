import { DecoratorFunction } from '@storybook/addons';
import React from 'react';

import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import UI from 'contexts/UI';

export const AuthDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(Auth.Provider, null, storyFn());
};

export const ClusterOverviewDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(ClusterOverview.Provider, null, storyFn());
};

export const UIDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(UI.Provider, null, storyFn());
};
