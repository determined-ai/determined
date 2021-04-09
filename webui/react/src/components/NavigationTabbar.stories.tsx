import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';
import RouterDecorator from 'storybook/RouterDecorator';
import StoreDecorator from 'storybook/StoreDecorator';

import NavigationTabbar from './NavigationTabbar';

export default {
  component: NavigationTabbar,
  decorators: [ StoreDecorator, RouterDecorator ],
  parameters: { layout: 'fullscreen' },
  title: 'NavigationTabbar',
};

const NavigationTabbarLoggedIn = () => {
  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return <NavigationTabbar />;
};

export const Default = (): React.ReactNode => (
  <div style={{ display: 'flex', flexDirection: 'column' }}>
    <div style={{ flexGrow: 1, height: 'calc(100vh - 56px)' }} />
    <NavigationTabbarLoggedIn />
  </div>
);
