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

import { RouteConfig } from './types';

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
