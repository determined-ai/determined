import React from 'react';

import ClusterLogs from 'pages/ClusterLogs';
import Clusters from 'pages/Clusters';
import Dashboard from 'pages/Dashboard';
import Default from 'pages/Default';
import Deprecated from 'pages/Deprecated';
import ExperimentDetails from 'pages/ExperimentDetails';
import InteractiveTask from 'pages/InteractiveTask';
import ModelDetails from 'pages/ModelDetails';
import ModelRegistry from 'pages/ModelRegistry';
import ModelVersionDetails from 'pages/ModelVersionDetails';
import ProjectDetailsWrapper from 'pages/ProjectDetailsWrapper';
import Reload from 'pages/Reload';
import ResourcepoolDetail from 'pages/ResourcepoolDetail';
import Settings from 'pages/Settings';
import SignIn from 'pages/SignIn';
import SignOut from 'pages/SignOut';
import TaskList from 'pages/TaskList';
import { TaskLogsWrapper } from 'pages/TaskLogs';
import TrialDetails from 'pages/TrialDetails';
import Wait from 'pages/Wait';
import Webhooks from 'pages/WebhookList';
import WorkspaceDetails from 'pages/WorkspaceDetails';
import WorkspaceList from 'pages/WorkspaceList';
import { RouteConfig } from 'shared/types';

import Routes from './routes';

const routeComponentMap: Record<string, React.ReactNode> = {
  cluster: <Deprecated />,
  clusterHistorical: <Deprecated />,
  clusterLogs: <ClusterLogs />,
  clusters: <Clusters />,
  dashboard: <Dashboard />,
  default: <Default />,
  experimentDetails: <ExperimentDetails />,
  interactive: <InteractiveTask />,
  jobs: <Deprecated />,
  modelDetails: <ModelDetails />,
  models: <ModelRegistry />,
  modelVersionDetails: <ModelVersionDetails />,
  projectDetails: <ProjectDetailsWrapper />,
  reload: <Reload />,
  resourcepool: <ResourcepoolDetail />,
  settings: <Settings />,
  signIn: <SignIn />,
  signOut: <SignOut />,
  taskList: <TaskList />,
  taskLogs: <TaskLogsWrapper />,
  trialDetails: <TrialDetails />,
  uncategorized: <ProjectDetailsWrapper />,
  wait: <Wait />,
  webhooks: <Webhooks />,
  workspaceDetails: <WorkspaceDetails />,
  workspaceList: <WorkspaceList />,
};

const defaultRouteId = 'uncategorized';
const rbacDefaultRouteId = 'workspaceList';
const dashboardDefaultRouteId = 'dashboard';

const appRoutes: RouteConfig[] = Routes.map((route) => {
  if (!routeComponentMap[route.id]) {
    throw new Error(`Missing route component for ${route.id}`);
  }
  return { ...route, element: routeComponentMap[route.id] };
});

export const defaultRoute = appRoutes.find((route) => route.id === defaultRouteId) as RouteConfig;
export const rbacDefaultRoute = appRoutes.find(
  (route) => route.id === rbacDefaultRouteId,
) as RouteConfig;
export const dashboardDefaultRoute = appRoutes.find(
  (route) => route.id === dashboardDefaultRouteId,
) as RouteConfig;

appRoutes.push({
  id: 'catch-all',
  path: '*',
  redirect: defaultRoute.path,
});

export default appRoutes;
