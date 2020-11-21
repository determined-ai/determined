import { DecoratorFunction } from '@storybook/addons';
import React from 'react';

import Auth from 'contexts/Auth';
import ClusterOverview from 'contexts/ClusterOverview';
import Info from 'contexts/Info';
import UI from 'contexts/UI';
import Users from 'contexts/Users';

export const AuthDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(Auth.Provider, null, storyFn());
};

export const ClusterOverviewDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(ClusterOverview.Provider, null, storyFn());
};

export const InfoDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(Info.Provider, null, storyFn());
};

export const UIDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(UI.Provider, null, storyFn());
};

export const UsersDecorator: DecoratorFunction<React.ReactNode> = storyFn => {
  return React.createElement(Users.Provider, null, storyFn());
};
