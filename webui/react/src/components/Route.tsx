import React, { ReactNode } from 'react';
import { Route as DomRoute, Redirect } from 'react-router-dom';

import Auth from 'contexts/Auth';
import { RouteConfigItem } from 'routes';
import { clone } from 'utils/data';

/*
 * A wrapper for <Route> that redirects to the login
 * screen if you're not yet authenticated.
 */

const Route: React.FC<RouteConfigItem> = ({ component, ...props }: RouteConfigItem) => {
  const auth = Auth.useStateContext();

  if (props.needAuth && !auth.isAuthenticated) {
    return <DomRoute
      {...props}
      render={(location): ReactNode => <Redirect to={{
        pathname: '/det/login',
        state: { loginRedirect: clone(location.location) },
      }} />}
    />;
  }

  return <DomRoute component={component} {...props} />;
};

export default Route;
