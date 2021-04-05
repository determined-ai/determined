import React, { useEffect } from 'react';

import Auth from 'contexts/Auth';
import { AuthDecorator, ClusterOverviewDecorator, UIDecorator } from 'storybook/ContextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';

import NavigationSideBar from './NavigationSideBar';

export default {
  component: NavigationSideBar,
  decorators: [ AuthDecorator, ClusterOverviewDecorator, RouterDecorator, UIDecorator ],
  title: 'NavigationSideBar',
};

const NavigationLoggedIn = () => {
  const setAuth = Auth.useActionContext();

  useEffect(() => {
    setAuth({ type: Auth.ActionType.Set, value: { isAuthenticated: true } });
  }, [ setAuth ]);

  return <NavigationSideBar />;
};

export const Default = (): React.ReactNode => (
  <div style={{ display: 'flex', width: '100vw' }}>
    <NavigationLoggedIn />;
    <div style={{ flexGrow: 1 }}>Content</div>
  </div>
);
