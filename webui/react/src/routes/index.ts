import { FC } from 'react';

import Cluster from 'pages/Cluster';
import ClusterLogs from 'pages/ClusterLogs';
import Clusters from 'pages/Clusters';
import Dashboard from 'pages/Dashboard';
import ExperimentDetails from 'pages/ExperimentDetails';
import ExperimentList from 'pages/ExperimentList';
import JobQueue from 'pages/JobQueue/JobQueue';
import ModelDetails from 'pages/ModelDetails';
import ModelRegistry from 'pages/ModelRegistry';
import ModelVersionDetails from 'pages/ModelVersionDetails';
import Reload from 'pages/Reload';
import ResourcepoolDetail from 'pages/ResourcepoolDetail';
import SignIn from 'pages/SignIn';
import SignOut from 'pages/SignOut';
import TaskList from 'pages/TaskList';
import TaskLogs from 'pages/TaskLogs';
import TrialDetails from 'pages/TrialDetails';
import Wait from 'pages/Wait';

import { RouteConfig } from '../shared/types';

import Routes from './routes';

const routeComponentMap: Record<string, FC> = {
  cluster: Cluster,
  clusterLogs: ClusterLogs,
  clusters: Clusters,
  dashboard: Dashboard,
  experimentDetails: ExperimentDetails,
  experimentList: ExperimentList,
  jobs: JobQueue,
  modelDetails: ModelDetails,
  models: ModelRegistry,
  modelVersionDetails: ModelVersionDetails,
  reload: Reload,
  resourcepool: ResourcepoolDetail,
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
