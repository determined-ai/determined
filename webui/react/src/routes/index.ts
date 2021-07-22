import { FC } from 'react';

import Cluster from 'pages/Cluster';
import Dashboard from 'pages/Dashboard';
import ExperimentDetails from 'pages/ExperimentDetails';
import ExperimentList from 'pages/ExperimentList';
import MasterLogs from 'pages/MasterLogs';
import Reload from 'pages/Reload';
import SignIn from 'pages/SignIn';
import SignOut from 'pages/SignOut';
import TaskList from 'pages/TaskList';
import TaskLogs from 'pages/TaskLogs';
import TrialDetails from 'pages/TrialDetails';
import Wait from 'pages/Wait';

import Routes from './routes';
import { RouteConfig } from './types';

const routeComponentMap: Record<string, FC> = {
  cluster: Cluster,
  dashboard: Dashboard,
  experimentDetails: ExperimentDetails,
  experimentList: ExperimentList,
  masterLogs: MasterLogs,
  reload: Reload,
  signIn: SignIn,
  signOut: SignOut,
  taskList: TaskList,
  taskLogs: TaskLogs,
  trialDetails: TrialDetails,
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
