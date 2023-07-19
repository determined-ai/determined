import React from 'react';

import Admin from 'pages/Admin';
import ClusterLogs from 'pages/ClusterLogs';
import Clusters from 'pages/Clusters';
import Dashboard from 'pages/Dashboard';
import DefaultRoute from 'pages/DefaultRoute';
import Deprecated from 'pages/Deprecated';
import ExperimentDetails from 'pages/ExperimentDetails';
import InteractiveTask from 'pages/InteractiveTask';
import ModelDetails from 'pages/ModelDetails';
import ModelRegistryPage from 'pages/ModelRegistryPage';
import ModelVersionDetails from 'pages/ModelVersionDetails';
import ProjectDetails from 'pages/ProjectDetails';
import Reload from 'pages/Reload';
import ResourcepoolDetail from 'pages/ResourcepoolDetail';
import SignIn from 'pages/SignIn';
import SignOut from 'pages/SignOut';
import TaskListPage from 'pages/TaskListPage';
import { TaskLogsWrapper } from 'pages/TaskLogs';
import TrialDetails from 'pages/TrialDetails';
import Wait from 'pages/Wait';
import Webhooks from 'pages/WebhookList';
import WorkspaceDetails from 'pages/WorkspaceDetails';
import WorkspaceList from 'pages/WorkspaceList';
import { RouteConfig } from 'types';

import Routes from './routes';

const routeComponentMap: Record<string, React.ReactNode> = {
  admin: <Admin />,
  cluster: <Deprecated />,
  clusterHistorical: <Deprecated />,
  clusterLogs: <ClusterLogs />,
  clusters: <Clusters />,
  dashboard: <Dashboard />,
  default: <DefaultRoute />,
  experimentDetails: <ExperimentDetails />,
  interactive: <InteractiveTask />,
  jobs: <Deprecated />,
  modelDetails: <ModelDetails />,
  models: <ModelRegistryPage />,
  modelVersionDetails: <ModelVersionDetails />,
  projectDetails: <ProjectDetails key="projectdetails" />,
  reload: <Reload />,
  resourcepool: <ResourcepoolDetail />,
  signIn: <SignIn />,
  signOut: <SignOut />,
  taskList: <TaskListPage />,
  taskLogs: <TaskLogsWrapper />,
  trialDetails: <TrialDetails />,
  uncategorized: <ProjectDetails key="uncategorized" />,
  wait: <Wait />,
  webhooks: <Webhooks />,
  workspaceDetails: <WorkspaceDetails />,
  workspaceList: <WorkspaceList />,
};

const defaultRouteId = 'default';
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
