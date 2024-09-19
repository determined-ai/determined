import React from 'react';

const Admin = React.lazy(() => import('pages/Admin'));
const Cluster = React.lazy(() => import('pages/Cluster'));
const ClusterLogs = React.lazy(() => import('pages/ClusterLogs'));
const Dashboard = React.lazy(() => import('pages/Dashboard'));
const DefaultRoute = React.lazy(() => import('pages/DefaultRoute'));
const Deprecated = React.lazy(() => import('pages/Deprecated'));
const ExperimentDetails = React.lazy(() => import('pages/ExperimentDetails'));
const InteractiveTask = React.lazy(() => import('pages/InteractiveTask'));
const ModelDetails = React.lazy(() => import('pages/ModelDetails'));
const ModelRegistryPage = React.lazy(() => import('pages/ModelRegistryPage'));
const ModelVersionDetails = React.lazy(() => import('pages/ModelVersionDetails'));
const ProjectDetails = React.lazy(() => import('pages/ProjectDetails'));
const Reload = React.lazy(() => import('pages/Reload'));
const ResourcepoolDetail = React.lazy(() => import('pages/ResourcePool/ResourcepoolDetail'));
const SearchDetails = React.lazy(() => import('pages/SearchDetails'));
const SignIn = React.lazy(() => import('pages/SignIn'));
const SignOut = React.lazy(() => import('pages/SignOut'));
const TaskListPage = React.lazy(() => import('pages/TaskListPage'));
const TaskLogsWrapper = React.lazy(() =>
  import('pages/TaskLogs').then((module) => ({ default: module.TaskLogsWrapper })),
);
const TemplatesPage = React.lazy(() => import('pages/Templates/TemplatesPage'));
const TrialDetails = React.lazy(() => import('pages/TrialDetails'));
const Wait = React.lazy(() => import('pages/Wait'));
const Webhooks = React.lazy(() => import('pages/WebhookList'));
const WorkspaceDetails = React.lazy(() => import('pages/WorkspaceDetails'));
const WorkspaceList = React.lazy(() => import('pages/WorkspaceList'));
import { RouteConfig } from 'types';

import Routes from './routes';

const routeComponentMap: Record<string, React.ReactNode> = {
  admin: <Admin />,
  cluster: <Deprecated />,
  clusterHistorical: <Deprecated />,
  clusterLogs: <ClusterLogs />,
  clusters: <Cluster />,
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
  searchDetails: <SearchDetails />,
  signIn: <SignIn />,
  signOut: <SignOut />,
  taskList: <TaskListPage />,
  taskLogs: <TaskLogsWrapper />,
  templates: <TemplatesPage />,
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
