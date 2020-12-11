import { FC } from 'react';

import Cluster from 'pages/Cluster';
import Dashboard from 'pages/Dashboard';
import ExperimentDetailPage from 'pages/ExperimentDetailPage';
import ExperimentDetails from 'pages/ExperimentDetails';
import ExperimentList from 'pages/ExperimentList';
import HGICluster from 'pages/HGICluster';
import MasterLogs from 'pages/MasterLogs';
import SignIn from 'pages/SignIn';
import SignOut from 'pages/SignOut';
import TaskList from 'pages/TaskList';
import TaskLogs from 'pages/TaskLogs';
import TrialDetails from 'pages/TrialDetails';
import TrialLogs from 'pages/TrialLogs';
import Wait from 'pages/Wait';

import Routes from './routes';
import { RouteConfig } from './types';

const routeComponentMap: Record<string, FC> = {
  cluster: Cluster,
  dashboard: Dashboard,
  experimentDetailPage: ExperimentDetailPage,
  experimentDetails: ExperimentDetails,
  experimentList: ExperimentList,
  hgiCluster: HGICluster,
  masterLogs: MasterLogs,
  signIn: SignIn,
  signOut: SignOut,
  taskList: TaskList,
  taskLogs: TaskLogs,
  trialDetails: TrialDetails,
  trialLogs: TrialLogs,
  wait: Wait,
};

const defaultRouteId = 'dashboard';

const appRoutes: RouteConfig[] = Routes.map(route => {
  if (!routeComponentMap[route.id]) throw new Error(`Missing route component for ${route.id}`);
  return {
    ...route,
    component: routeComponentMap[route.id],
  };
});

export const defaultRoute = appRoutes
  .find(route => route.id === defaultRouteId) as RouteConfig;

appRoutes.push({
  id: 'catch-all',
  path: '*',
  redirect: defaultRoute.path,
});

export default appRoutes;
