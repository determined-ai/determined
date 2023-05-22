import React from 'react';
import { Navigate } from 'react-router-dom';

import { dashboardDefaultRoute, rbacDefaultRoute } from 'routes';
import determinedStore from 'stores/determinedInfo';
import { useObservable } from 'utils/observable';

const Default: React.FC = () => {
  const { rbacEnabled } = useObservable(determinedStore.info);

  if (rbacEnabled) {
    return <Navigate to={rbacDefaultRoute.path} />;
  } else {
    return <Navigate to={dashboardDefaultRoute.path} />;
  }
};

export default Default;
