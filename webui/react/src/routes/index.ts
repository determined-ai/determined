import { RouteProps } from 'react-router';

import Authentication from 'pages/Authentication';
import Dashboard from 'pages/Dashboard';
import Determined from 'pages/Determined';
import history from 'routes/history';
import { isFullPath, parseUrl } from 'utils/routes';

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
  const pathname = parseUrl(url).pathname;
  return !!appRoutes.find(route => pathname.startsWith(route.path));
};

export const routeToExternalUrl = (path: string): void => {
  if (!isFullPath(path)) {
    const pathPrefix = process.env.IS_DEV ? 'http://localhost:8080' : '';
    path = `${pathPrefix}${path}`;
  }
  window.location.assign(path);
};

export const routeAll = (path: string): void => {
  if (!isReactRoute(path)) {
    routeToExternalUrl(path);
  } else {
    history.push(path);
  }
};
