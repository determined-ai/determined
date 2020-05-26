import React from 'react';

import { appRoutes, RouteConfigItem } from 'routes';

import Breadcrumb from './Breadcrumb';

export default {
  component: Breadcrumb,
  title: 'Breadcrumb',
};

export const Default = (): React.ReactNode =>
  <Breadcrumb route={appRoutes.find(route => route.id === 'dashboard') as RouteConfigItem} />;
export const Experiments = (): React.ReactNode =>
  <Breadcrumb route={{ path: '/det/dashboard/experiments' } as RouteConfigItem} />;
export const ExperimentDetail = (): React.ReactNode =>
  <Breadcrumb route={{ path: '/det/dashboard/experiments/1' } as RouteConfigItem} />;
