import React, { useEffect } from 'react';

import { StoreActionType, useStoreDispatch } from 'contexts/Store';
import RouterDecorator from 'storybook/RouterDecorator';
import StoreDecorator from 'storybook/StoreDecorator';

import NavigationTopbar from './NavigationTopbar';

export default {
  component: NavigationTopbar,
  decorators: [ StoreDecorator, RouterDecorator ],
  parameters: { layout: 'fullscreen' },
  title: 'NavigationTopbar',
};

const NavigationTopbarLoggedIn = () => {
  const storeDispatch = useStoreDispatch();

  useEffect(() => {
    storeDispatch({ type: StoreActionType.SetAuth, value: { isAuthenticated: true } });
  }, [ storeDispatch ]);

  return <NavigationTopbar />;
};

export const Default = (): React.ReactNode => (
  <NavigationTopbarLoggedIn />
);
