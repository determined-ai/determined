import { FC } from 'react';

import Cluster from 'pages/Cluster';
import ClusterLogs from 'pages/ClusterLogs';
import Clusters from 'pages/Clusters';
import ExperimentDetails from 'pages/ExperimentDetails';
import InteractiveTask from 'pages/InteractiveTask';
import JobQueue from 'pages/JobQueue/JobQueue';
import ModelDetails from 'pages/ModelDetails';
import ModelRegistry from 'pages/ModelRegistry';
import ModelVersionDetails from 'pages/ModelVersionDetails';
import ProjectDetails from 'pages/ProjectDetails';
import Reload from 'pages/Reload';
import ResourcepoolDetail from 'pages/ResourcepoolDetail';
import SignIn from 'pages/SignIn';
import SignOut from 'pages/SignOut';
import TaskList from 'pages/TaskList';
import { TaskLogsWrapper } from 'pages/TaskLogs';
import TrialDetails from 'pages/TrialDetails';
import Wait from 'pages/Wait';
import WorkspaceDetails from 'pages/WorkspaceDetails';
import WorkspaceList from 'pages/WorkspaceList';

import { RouteConfig } from '../shared/types';

import Routes from './routes';

const routeComponentMap: Record<string, FC> = {
  cluster: Cluster,
  clusterLogs: ClusterLogs,
  clusters: Clusters,
  experimentDetails: ExperimentDetails,
  interactive: InteractiveTask,
  jobs: JobQueue,
  modelDetails: ModelDetails,
  models: ModelRegistry,
  modelVersionDetails: ModelVersionDetails,
  projectDetails: ProjectDetails,
  reload: Reload,
  resourcepool: ResourcepoolDetail,
  signIn: SignIn,
  signOut: SignOut,
  taskList: TaskList,
  taskLogs: TaskLogsWrapper,
  trialDetails: TrialDetails,
  uncategorized: ProjectDetails,
  wait: Wait,
  workspaceDetails: WorkspaceDetails,
  workspaceList: WorkspaceList,
};

const defaultRouteId = 'uncategorized';

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
