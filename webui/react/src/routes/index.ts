import { RouteProps } from 'react-router';

import Authentication from 'pages/Authentication';
import Dashboard from 'pages/Dashboard';
import Determined from 'pages/Determined';

/*
 * Router Configuration
 * If the component is not defined, the route is assumed to be an external route,
 * meaning React will attempt to load the path outside of the internal routing
 * mechanism.
 */
export interface RouteConfigItem extends RouteProps {
  id: string;
  icon?: string;
  path: string;
  popout?: boolean;
  suffixIcon?: string;
  title: string;
  needAuth?: boolean;
}

export const isFullPath = (path: string): boolean => {
  return path.startsWith('http');
};

export const isCrossoverRoute = (path: string): boolean => {
  return path.startsWith('/ui') || path.includes(':8080/ui');
};

export const crossoverRoute = (path: string): void => {
  if (!isFullPath(path)) {
    const pathPrefix = process.env.IS_DEV ? 'http://localhost:8080' : '';
    path = `${pathPrefix}${path}`;
  }
  window.location.assign(path);
};

export const appRoutes: RouteConfigItem[] = [
  {
    component: Determined,
    id: 'det',
    needAuth: true,
    path: '/det/dashboard',
    title: 'Determined',
  },
  {
    component: Authentication,
    id: 'login',
    needAuth: false,
    path: '/det/login',
    title: 'Login',
  },
  {
    component: Authentication,
    id: 'logout',
    needAuth: false,
    path: '/det/logout',
    title: 'Logout',
  },
  {
    id: 'docs',
    path: '/docs/',
    popout: true,
    suffixIcon: 'popout',
    title: 'Docs',
  },
];
export const defaultAppRouteId = appRoutes[0].id;

export const detRoutes: RouteConfigItem[] = [
  {
    component: Dashboard,
    icon: 'user',
    id: 'dashboard',
    needAuth: true,
    path: '/det/dashboard',
    title: 'Dashboard',
  },
  {
    icon: 'experiment',
    id: 'experiments',
    path: '/ui/experiments',
    title: 'Experiments',
  },
  {
    icon: 'notebook',
    id: 'notebooks',
    path: '/ui/notebooks',
    title: 'Notebooks',
  },
  {
    icon: 'tensorboard',
    id: 'tensorboards',
    path: '/ui/tensorboards',
    title: 'TensorBoards',
  },
  {
    icon: 'cluster',
    id: 'cluster',
    path: '/ui/cluster',
    title: 'Cluster',
  },
  {
    icon: 'shell',
    id: 'shells',
    path: '/ui/shells',
    title: 'Shells',
  },
  {
    icon: 'command',
    id: 'commands',
    path: '/ui/commands',
    title: 'Commands',
  },
];
export const defaultDetRouteId = detRoutes[0].id;
