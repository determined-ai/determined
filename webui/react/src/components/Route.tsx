import React from 'react';
import { Route as DomRoute, Redirect } from 'react-router-dom';

import Auth from 'contexts/Auth';
import { RouteConfigItem } from 'routes';

/*
 * A wrapper for <Route> that redirects to the login
 * screen if you're not yet authenticated.
 */

const Route: React.FC<RouteConfigItem> = (props: RouteConfigItem) => {
  const auth = Auth.useStateContext();

  if (props.needAuth && !auth.isAuthenticated) {
    return <Redirect to={ `/det/login?redirect=${props.path}` } />;
  }

  return <DomRoute {...props} />;
};

export default Route;
