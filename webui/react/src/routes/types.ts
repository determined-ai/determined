import { RouteProps } from 'react-router';

/*
 * Router Configuration
 * If the component is not defined, the route is assumed to be an external route,
 * meaning React will attempt to load the path outside of the internal routing
 * mechanism.
 */
export interface RouteConfig extends RouteProps {
  icon?: string;
  id: string;
  needAuth?: boolean;
  path: string;
  popout?: boolean;
  redirect?: string;
  suffixIcon?: string;
  title?: string;
}
