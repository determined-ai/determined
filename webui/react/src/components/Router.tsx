import React, { ReactNode, useEffect } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import Auth from 'contexts/Auth';
import UI from 'contexts/UI';
import useAuthCheck from 'hooks/useAuthCheck';
import { RouteConfig } from 'routes/types';
import { clone } from 'utils/data';

interface Props {
  routes: RouteConfig[];
}

const Router: React.FC<Props> = (props: Props) => {
  const auth = Auth.useStateContext();
  const checkAuth = useAuthCheck();
  const setUI = UI.useActionContext();

  useEffect(() => {
    checkAuth();
  }, [ checkAuth ]);

  useEffect(() => {
    if (auth.isAuthenticated) {
      setUI({ type: UI.ActionType.HideSpinner });
    }
  }, [ auth.isAuthenticated, setUI ]);

  return (
    <Switch>
      {props.routes.map(config => {
        const { component, ...route } = config;

        if (route.needAuth && !auth.isAuthenticated) {
          // Do not mount login page until auth is checked.
          if (!auth.checked) return <Route key={route.id} {...route} />;
          return <Route
            key={route.id}
            {...route}
            render={({ location }): ReactNode => <Redirect to={{
              pathname: '/login',
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
