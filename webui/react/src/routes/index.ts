import { FC } from 'react';

import ClusterLogs from 'pages/ClusterLogs';
import Clusters from 'pages/Clusters';
import Deprecated from 'pages/Deprecated';
import ExperimentComparison from 'pages/ExperimentComparison';
import ExperimentDetails from 'pages/ExperimentDetails';
import InteractiveTask from 'pages/InteractiveTask';
import ModelDetails from 'pages/ModelDetails';
import ModelRegistry from 'pages/ModelRegistry';
import ModelVersionDetails from 'pages/ModelVersionDetails';
import ProjectDetails from 'pages/ProjectDetails';
import Reload from 'pages/Reload';
import ResourcepoolDetail from 'pages/ResourcepoolDetail';
import Settings from 'pages/Settings';
import SignIn from 'pages/SignIn';
import SignOut from 'pages/SignOut';
import TaskList from 'pages/TaskList';
import { TaskLogsWrapper } from 'pages/TaskLogs';
import TrialDetails from 'pages/TrialDetails';
import Wait from 'pages/Wait';
import WorkspaceDetails from 'pages/WorkspaceDetails';
import WorkspaceList from 'pages/WorkspaceList';
import { RouteConfig } from 'shared/types';

import Routes from './routes';

const routeComponentMap: Record<string, FC> = {
  cluster: Deprecated,
  clusterHistorical: Deprecated,
  clusterLogs: ClusterLogs,
  clusters: Clusters,
  experimentComparison: ExperimentComparison,
  experimentDetails: ExperimentDetails,
  interactive: InteractiveTask,
  jobs: Deprecated,
  modelDetails: ModelDetails,
  models: ModelRegistry,
  modelVersionDetails: ModelVersionDetails,
  projectDetails: ProjectDetails,
  reload: Reload,
  resourcepool: ResourcepoolDetail,
  settings: Settings,
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

const appRoutes: RouteConfig[] = Routes.map((route) => {
  if (!routeComponentMap[route.id]) throw new Error(`Missing route component for ${route.id}`);
  return {
    ...route,
    component: routeComponentMap[route.id],
  };
});

export const defaultRoute = appRoutes
  .find((route) => route.id === defaultRouteId) as RouteConfig;

appRoutes.push({
  id: 'catch-all',
  path: '*',
  redirect: defaultRoute.path,
});

export default appRoutes;
