import React, { useEffect } from 'react';

import { useAuth } from 'stores/auth';

import NavigationTabbar from './NavigationTabbar';

export default {
  component: NavigationTabbar,
  parameters: { layout: 'fullscreen' },
  title: 'Determined/Navigation/NavigationTabbar',
};

const NavigationTabbarLoggedIn = () => {
  const { setAuth } = useAuth();

  useEffect(() => {
    setAuth({ isAuthenticated: true });
  }, [setAuth]);

  return <NavigationTabbar />;
};

export const Default = (): React.ReactNode => (
  <div style={{ display: 'flex', flexDirection: 'column' }}>
    <div style={{ flexGrow: 1, height: 'calc(100vh - 56px)' }} />
    <NavigationTabbarLoggedIn />
  </div>
);
