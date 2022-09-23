import { Meta } from '@storybook/react';
import React, { useEffect } from 'react';

import { StoreAction, useStoreDispatch } from 'contexts/Store';

import NavigationSideBar from './NavigationSideBar';

export default {
  component: NavigationSideBar,
  title: 'Determined/Navigation/NavigationSideBar',
} as Meta<typeof NavigationSideBar>;

const NavigationLoggedIn = () => {
  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreAction.SetAuth, value: { isAuthenticated: true } });
  }, [storeDispatch]);

  return <NavigationSideBar />;
};

export const Default = (): React.ReactNode => (
  <div style={{ display: 'flex', width: '100vw' }}>
    <NavigationLoggedIn />
  </div>
);
