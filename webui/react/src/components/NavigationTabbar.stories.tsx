import React, { useEffect } from 'react';

import Auth from 'contexts/Auth';
import { AuthDecorator, ClusterOverviewDecorator, UIDecorator } from 'storybook/ContextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';

import NavigationTabbar from './NavigationTabbar';

export default {
  component: NavigationTabbar,
  decorators: [
    AuthDecorator,
    ClusterOverviewDecorator,
    RouterDecorator,
    UIDecorator ],
  parameters: { layout: 'fullscreen' },
  title: 'NavigationTabbar',
};

const NavigationTabbarLoggedIn = () => {
  const setAuth = Auth.useActionContext();

  useEffect(() => {
    setAuth({ type: Auth.ActionType.Set, value: { isAuthenticated: true } });
  }, [ setAuth ]);

  return <NavigationTabbar />;
};

export const Default = (): React.ReactNode => (
  <div style={{ display: 'flex', flexDirection: 'column' }}>
    <div style={{ flexGrow: 1, height: 'calc(100vh - 56px)' }} />
    <NavigationTabbarLoggedIn />
  </div>
);
