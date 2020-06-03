import { pathToRegexp } from 'path-to-regexp';
import { RouteProps } from 'react-router';

import Dashboard from 'pages/Dashboard';
import SignIn from 'pages/SignIn';
import SignOut from 'pages/SignOut';
import history from 'routes/history';
import { clone } from 'utils/data';
import { ensureAbsolutePath, isFullPath, parseUrl } from 'utils/routes';

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

const defaultPath = '/det/dashboard';
const dashboardRoute =
  {
    component: Dashboard,
    icon: 'user',
    id: 'dashboard',
    needAuth: true,
    path: defaultPath,
    title: 'Dashboard',
  };

export const appRoutes: RouteConfig[] = [
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
    redirect: defaultPath,
  },
];

export const defaultAppRoute = dashboardRoute;

export const sidebarRoutes: RouteConfig[] = [
  dashboardRoute,
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
export const defaultSideBarRoute = sidebarRoutes[0];

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

// to support running the SPA off of a separate port from the cluster and have the links to Elm
// SPA work.
export const setupUrlForDev = (path: string): string => {
  if (process.env.IS_DEV && !isFullPath(path) && isDetRoute(path) && !isReactRoute(path)) {
    const pathPrefix = process.env.IS_DEV ? 'http://localhost:8080' : '';
    return pathPrefix + ensureAbsolutePath(path);
  }
  return path;
};

export const routeToExternalUrl = (path: string): void => {
  window.location.assign(setupUrlForDev(path));
};

export const routeAll = (path: string): void => {
  if (!isReactRoute(path)) {
    routeToExternalUrl(path);
  } else {
    history.push(path, { loginRedirect: clone(window.location) });
  }
};
