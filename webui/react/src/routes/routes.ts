import { RouteConfig } from './types';

const routes: RouteConfig[] = [
  {
    id: 'trialDetails',
    needAuth: true,
    path: '/experiments/:experimentId/trials/:trialId/:tab?',
    title: 'Trial',
  },
  {
    id: 'trialDetails',
    needAuth: true,
    path: '/trials/:trialId/:tab?',
    title: 'Trial',
  },
  {
    id: 'experimentDetails',
    needAuth: true,
    path: '/experiments/:experimentId/:tab/:viz',
    title: 'Experiment',
  },
  {
    id: 'experimentDetails',
    needAuth: true,
    path: '/experiments/:experimentId/:tab',
    title: 'Experiment',
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
    path: '/cluster/:tab?',
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
  {
    id: 'reload',
    needAuth: false,
    path: '/reload',
    title: 'Reload',
  },
];

export default routes;
