import React from 'react';
import { Navigate } from 'react-router-dom';

import useFeature from 'hooks/useFeature';
import { dashboardDefaultRoute, defaultRoute, rbacDefaultRoute } from 'routes';

const Default: React.FC = () => {
  const dashboardEnabled = useFeature().isOn('dashboard');
  const rbacEnabled = useFeature().isOn('rbac');

  if (dashboardEnabled) {
    return <Navigate to={dashboardDefaultRoute.path} />;
  } else if (rbacEnabled) {
    return <Navigate to={rbacDefaultRoute.path} />;
  } else {
    return <Navigate to={defaultRoute.path} />;
  }
};

export default Default;
