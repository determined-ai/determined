import React, { useEffect } from 'react';

import Auth from 'contexts/Auth';
import { AuthDecorator, ClusterOverviewDecorator, UIDecorator } from 'storybook/ContextDecorators';
import RouterDecorator from 'storybook/RouterDecorator';

import NavigationTabbar from './NavigationTabbar';

export default {
  component: NavigationTabbar,
  decorators: [ AuthDecorator, ClusterOverviewDecorator, RouterDecorator, UIDecorator ],
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
  <div style={{
    border: 'solid 1px #cccccc',
    display: 'flex',
    flexDirection: 'column',
    height: 480,
    position: 'relative',
    width: 320,
  }}>
    <div style={{
      alignItems: 'center',
      display: 'flex',
      flexGrow: 1,
      justifyContent: 'center',
    }}>Content Area</div>
    <NavigationTabbarLoggedIn />
  </div>
);
