import { pathToRegexp } from 'path-to-regexp';
import { RouteProps } from 'react-router';

import Cluster from 'pages/Cluster';
import Dashboard from 'pages/Dashboard';
import ExperimentDetails from 'pages/ExperimentDetails';
import ExperimentList from 'pages/ExperimentList';
import MasterLogs from 'pages/MasterLogs';
import SignIn from 'pages/SignIn';
import SignOut from 'pages/SignOut';
import TaskList from 'pages/TaskList';
import TaskLogs from 'pages/TaskLogs';
import TrialDetails from 'pages/TrialDetails';
import TrialLogs from 'pages/TrialLogs';
import history from 'routes/history';
import { clone } from 'utils/data';
import { isFullPath, parseUrl } from 'utils/routes';

/*
 * Router Configuration
 * If the component is not defined, the route is assumed to be an external route,
 * meaning React will attempt to load the path outside of the internal routing
 * mechanism.
 */
export interface RouteConfig extends RouteProps {
  id: string;
  icon?: string;
  path: string;
  popout?: boolean;
  redirect?: string;
  suffixIcon?: string;
  title?: string;
  needAuth?: boolean;
}

const dashboardRoute = {
  component: Dashboard,
  icon: 'user',
  id: 'dashboard',
  needAuth: true,
  path: '/det/dashboard',
  title: 'Dashboard',
};

const clusterRoute = {
  component: Cluster,
  icon: 'cluster',
  id: 'cluster-det',
  needAuth: true,
  path: '/det/cluster',
  title: 'Cluster',
};

const experimentListRoute = {
  component: ExperimentList,
  icon: 'experiment',
  id: 'experimentList',
  needAuth: true,
  path: '/det/experiments',
  title: 'Experiments',
};

const experimentDetailsRoute = {
  component: ExperimentDetails,
  id: 'experimentDetails',
  needAuth: true,
  path: '/det/experiments/:experimentId',
  title: 'Experiment',
};

const trialDetailsRoute = {
  component: TrialDetails,
  id: 'trialDetails',
  needAuth: true,
  path: '/det/trials/:trialId',
  title: 'Trial',
};

const taskLogsRoute = {
  component: TaskLogs,
  icon: 'logs',
  id: 'task-logs',
  needAuth: true,
  path: '/det/:taskType/:taskId/logs',
  title: 'Task Logs',
};

const taskListRoute = {
  component: TaskList,
  icon: 'tasks',
  id: 'taskList',
  needAuth: true,
  path: '/det/tasks',
  title: 'Tasks',
};

const masterLogsRoute = {
  component: MasterLogs,
  icon: 'logs',
  id: 'logs',
  needAuth: true,
  path: '/det/logs',
  title: 'Master Logs',
};

const trialLogsRoute = {
  component: TrialLogs,
  id: 'trial-logs',
  needAuth: true,
  path: '/det/trials/:trialId/logs',
  title: 'Trial Logs',
};

export const defaultAppRoute = dashboardRoute;

export const appRoutes: RouteConfig[] = [
  trialLogsRoute,
  trialDetailsRoute,
  experimentDetailsRoute,
  taskLogsRoute,
  taskListRoute,
  experimentListRoute,
  clusterRoute,
  masterLogsRoute,
  dashboardRoute,
  {
    component: SignIn,
    id: 'login',
    needAuth: false,
    path: '/det/login',
    title: 'Login',
  },
  {
    component: SignOut,
    id: 'logout',
    needAuth: false,
    path: '/det/logout',
    title: 'Logout',
  },
  {
    id: 'catch-all',
    path: '*',
    redirect: defaultAppRoute.path,
  },
];

// Add pages we don't want to expose to the public yet.
if (process.env.IS_DEV) {
  // appRoutes.push();
}

// Is the path going to be served from the same host?
const isDetRoute = (url: string): boolean => {
  if (!isFullPath(url)) return true;
  if (process.env.IS_DEV) {
    // dev live is served on a different port
    return parseUrl(url).hostname === window.location.hostname;
  }
  return parseUrl(url).host === window.location.host;
};

const isReactRoute = (url: string): boolean => {
  if (!isDetRoute(url)) return false;

  // Check to see if the path matches any of the defined app routes.
  const pathname = parseUrl(url).pathname;
  return !!appRoutes
    .filter(route => route.path !== '*')
    .find(route => {
      return route.exact ? pathname === route.path : !!pathToRegexp(route.path).exec(pathname);
    });
};

export const routeToExternalUrl = (path: string): void => {
  window.location.assign(path);
};

export const routeAll = (path: string): void => {
  if (!isReactRoute(path)) {
    routeToExternalUrl(path);
  } else {
    history.push(path, { loginRedirect: clone(window.location) });
  }
};
