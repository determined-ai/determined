import React, { useEffect } from 'react';

import { StoreActionType, useStoreDispatch } from 'contexts/Store';
import RouterDecorator from 'storybook/RouterDecorator';
import StoreDecorator from 'storybook/StoreDecorator';

import NavigationSideBar from './NavigationSideBar';

export default {
  component: NavigationSideBar,
  decorators: [ StoreDecorator, RouterDecorator ],
  title: 'NavigationSideBar',
};

const NavigationLoggedIn = () => {
  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreActionType.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return <NavigationSideBar />;
};

export const Default = (): React.ReactNode => (
  <div style={{ display: 'flex', width: '100vw' }}>
    <NavigationLoggedIn />;
    <div style={{ flexGrow: 1 }}>Content</div>
  </div>
);
