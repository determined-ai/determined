import { RouteConfig } from './types';

const routes: RouteConfig[] = [
  {
    id: 'trialLogs',
    needAuth: true,
    path: '/det/trials/:trialId/logs',
    title: 'Trial Logs',
  },
  {
    id: 'trialDetails',
    needAuth: true,
    path: '/det/trials/:trialId',
    title: 'Trial',
  },
  {
    id: 'experimentDetails',
    needAuth: true,
    path: '/det/experiments/:experimentId',
    title: 'Experiment',
  },
  {
    icon: 'logs',
    id: 'taskLogs',
    needAuth: true,
    path: '/det/:taskType/:taskId/logs',
    title: 'Task Logs',
  },
  {
    icon: 'tasks',
    id: 'taskList',
    needAuth: true,
    path: '/det/tasks',
    title: 'Tasks',
  },
  {
    icon: 'experiment',
    id: 'experimentList',
    needAuth: true,
    path: '/det/experiments',
    title: 'Experiments',
  },
  {
    icon: 'cluster',
    id: 'cluster',
    needAuth: true,
    path: '/det/cluster',
    title: 'Cluster',
  },
  {
    icon: 'logs',
    id: 'masterLogs',
    needAuth: true,
    path: '/det/logs',
    title: 'Master Logs',
  },
  {
    icon: 'user',
    id: 'dashboard',
    needAuth: true,
    path: '/det/dashboard',
    title: 'Dashboard',
  },
  {
    id: 'signIn',
    needAuth: false,
    path: '/det/login',
    title: 'Login',
  },
  {
    id: 'signOut',
    needAuth: false,
    path: '/det/logout',
    title: 'Logout',
  },
];

export default routes;
