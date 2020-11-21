import React, { useEffect } from 'react';

import Auth from 'contexts/Auth';
import { AuthDecorator, ClusterOverviewDecorator, UIDecorator } from 'storybook/ContextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';

import NavigationTopbar from './NavigationTopbar';

export default {
  component: NavigationTopbar,
  decorators: [
    AuthDecorator,
    ClusterOverviewDecorator,
    RouterDecorator,
    UIDecorator ],
  parameters: { layout: 'fullscreen' },
  title: 'NavigationTopbar',
};

const NavigationTopbarLoggedIn = () => {
  const setAuth = Auth.useActionContext();

  useEffect(() => {
    setAuth({ type: Auth.ActionType.Set, value: { isAuthenticated: true } });
  }, [ setAuth ]);

  return <NavigationTopbar />;
};

export const Default = (): React.ReactNode => (
  <NavigationTopbarLoggedIn />
);
