import { RouteConfig } from './types';

const routes: RouteConfig[] = [
  {
    id: 'trialLogs',
    needAuth: true,
    path: '/experiments/:experimentId/trials/:trialId/logs',
    title: 'Trial Logs',
  },
  {
    id: 'trialLogs',
    needAuth: true,
    path: '/trials/:trialId/logs',
    title: 'Trial Logs',
  },
  {
    id: 'trialDetails',
    needAuth: true,
    path: '/experiments/:experimentId/trials/:trialId',
    title: 'Trial',
  },
  {
    id: 'trialDetails',
    needAuth: true,
    path: '/trials/:trialId',
    title: 'Trial',
  },
  {
    id: 'experimentDetails',
    needAuth: true,
    path: '/experiments/:experimentId',
    title: 'Experiment',
  },
  {
    icon: 'logs',
    id: 'taskLogs',
    needAuth: true,
    path: '/:taskType/:taskId/logs',
    title: 'Task Logs',
  },
  {
    icon: 'tasks',
    id: 'taskList',
    needAuth: true,
    path: '/tasks',
    title: 'Tasks',
  },
  {
    icon: 'experiment',
    id: 'experimentList',
    needAuth: true,
    path: '/experiments',
    title: 'Experiments',
  },
  {
    icon: 'cluster',
    id: 'cluster',
    needAuth: true,
    path: '/cluster',
    title: 'Cluster',
  },
  {
    icon: 'logs',
    id: 'masterLogs',
    needAuth: true,
    path: '/logs',
    title: 'Master Logs',
  },
  {
    icon: 'user',
    id: 'dashboard',
    needAuth: true,
    path: '/dashboard',
    title: 'Dashboard',
  },
  {
    id: 'wait',
    needAuth: true,
    path: '/wait/:taskType/:taskId',
  },
  {
    id: 'signIn',
    needAuth: false,
    path: '/login',
    title: 'Login',
  },
  {
    id: 'signOut',
    needAuth: false,
    path: '/logout',
    title: 'Logout',
  },
];

export default routes;
