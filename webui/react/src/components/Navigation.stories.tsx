import React, { useEffect } from 'react';

import Auth from 'contexts/Auth';
import { AuthDecorator, ClusterOverviewDecorator, UIDecorator } from 'storybook/ContextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';

import Navigation from './Navigation';

export default {
  component: Navigation,
  decorators: [ AuthDecorator, ClusterOverviewDecorator, RouterDecorator, UIDecorator ],
  title: 'Navigation',
};

const NavigationLoggedIn = () => {
  const setAuth = Auth.useActionContext();

  useEffect(() => {
    setAuth({ type: Auth.ActionType.Set, value: { isAuthenticated: true } });
  }, [ setAuth ]);

  return <Navigation />;
};

export const Default = (): React.ReactNode => (
  <div style={{ display: 'flex', width: '100vw' }}>
    <NavigationLoggedIn />;
    <div style={{ flexGrow: 1 }}>Content</div>
  </div>
);
