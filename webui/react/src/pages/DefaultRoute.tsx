import React from 'react';
import { Navigate } from 'react-router-dom';

import useFeature from 'hooks/useFeature';
import { dashboardDefaultRoute, rbacDefaultRoute } from 'routes';

const Default: React.FC = () => {
  const rbacEnabled = useFeature().isOn('rbac');

  if (rbacEnabled) {
    return <Navigate to={rbacDefaultRoute.path} />;
  } else {
    return <Navigate to={dashboardDefaultRoute.path} />;
  }
};

export default Default;
