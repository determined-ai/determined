import { Meta } from '@storybook/react';
import React, { useEffect } from 'react';

import { useAuth } from 'stores/auth';

import NavigationSideBar from './NavigationSideBar';

export default {
  component: NavigationSideBar,
  title: 'Determined/Navigation/NavigationSideBar',
} as Meta<typeof NavigationSideBar>;

const NavigationLoggedIn = () => {
  const { setAuth } = useAuth();

  useEffect(() => {
    setAuth({ isAuthenticated: true });
  }, [setAuth]);

  return <NavigationSideBar />;
};

export const Default = (): React.ReactNode => (
  <div style={{ display: 'flex', width: '100vw' }}>
    <NavigationLoggedIn />
  </div>
);
