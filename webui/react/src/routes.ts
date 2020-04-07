import Dashboard from 'pages/Dashboard';
import Determined from 'pages/Determined';

/*
 * Router Configuration
 * If the component is not defined, the route is assumed to be an external route,
 * meaning React will attempt to load the path outside of the internal routing
 * mechanism.
 */
export interface RouteConfigItem {
  id: string;
  icon?: string;
  path: string;
  popout?: boolean;
  suffixIcon?: string;
  title: string;
  component?: React.FC;
  needAuth?: boolean;
}

export const crossoverRoute = (path: string): void => {
  const pathPrefix = process.env.IS_DEV ? 'http://localhost:8080' : '';
  window.location.assign(`${pathPrefix}${path}`);
};

export const appRoutes: RouteConfigItem[] = [
  {
    component: Determined,
    id: 'det',
    needAuth: true,
    path: '/det',
    title: 'Determined',
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
    icon: 'star',
    id: 'dashboard',
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
