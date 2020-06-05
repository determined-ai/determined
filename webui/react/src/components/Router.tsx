import React, { ReactNode } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

// import Route from 'components/Route';
import Auth from 'contexts/Auth';
import { RouteConfig } from 'routes';
import { clone } from 'utils/data';

interface Props {
  routes: RouteConfig[];
}

const Router: React.FC<Props> = (props: Props) => {
  const auth = Auth.useStateContext();

  return (
    <Switch>
      {props.routes.map(config => {
        const { component, ...route } = config;

        if (route.needAuth && !auth.isAuthenticated) {
          return <Route
            key={route.id}
            {...route}
            render={({ location }): ReactNode => <Redirect to={{
              pathname: '/det/login',
              state: { loginRedirect: clone(location) },
            }} />}
          />;
        } else if (route.redirect) {
          /*
          * We treat '*' as a catch-all path and specifically avoid wrapping the
          * `Redirect` with a `DomRoute` component. This ensures the catch-all
          * redirect will occur when encountered in the `Switch` traversal.
          */
          if (route.path === '*') {
            return <Redirect key={route.id} to={route.redirect} />;
          } else {
            return (
              <Route exact={route.exact} key={route.id} path={route.path}>
                <Redirect to={route.redirect} />;
              </Route>
            );
          }
        }

        return <Route component={component} key={route.id} {...route} />;
      })}
    </Switch>
  );
};

export default Router;
