import React, { ReactNode, useEffect, useState } from 'react';
import { Redirect, Route, Switch } from 'react-router-dom';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import useAuthCheck from 'hooks/useAuthCheck';
import { RouteConfig } from 'routes/types';
import { paths } from 'routes/utils';
import { clone } from 'utils/data';

interface Props {
  routes: RouteConfig[];
}

const Router: React.FC<Props> = (props: Props) => {
  const { auth } = useStore();
  const storeDispatch = useStoreDispatch();
  const [ canceler ] = useState(new AbortController());
  const checkAuth = useAuthCheck(canceler);

  useEffect(() => {
    checkAuth();
  }, [ checkAuth ]);

  useEffect(() => {
    if (auth.isAuthenticated) {
      storeDispatch({ type: StoreAction.HideUISpinner });
    }
  }, [ auth.isAuthenticated, storeDispatch ]);

  useEffect(() => {
    return () => canceler.abort();
  }, [ canceler ]);

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
              pathname: paths.login(),
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
