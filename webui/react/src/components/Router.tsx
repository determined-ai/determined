import React, { ReactNode, useEffect, useState } from 'react';
import { Redirect, Switch } from 'react-router-dom';
import { CompatRoute } from 'react-router-dom-v5-compat';

import { StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import useAuthCheck from 'hooks/useAuthCheck';
import { paths } from 'routes/utils';
import { RouteConfig } from 'shared/types';
import { filterOutLoginLocation } from 'shared/utils/routes';

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
      {props.routes.map((config) => {
        const { component, ...route } = config;

        if (route.needAuth && !auth.isAuthenticated) {
          // Do not mount login page until auth is checked.
          if (!auth.checked) return <CompatRoute {...route} key={route.id} />;
          return (
            <CompatRoute
              {...route}
              key={route.id}
              render={({ location }: {location: Location}): ReactNode => (
                <Redirect
                  to={{
                    pathname: paths.login(),
                    state: { loginRedirect: filterOutLoginLocation(location) },
                  }}
                />
              )}
            />
          );
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
              <CompatRoute exact={route.exact} key={route.id} path={route.path}>
                <Redirect to={route.redirect} />;
              </CompatRoute>
            );
          }
        }

        return <CompatRoute {...route} component={component} key={route.id} />;
      })}
    </Switch>
  );
};

export default Router;
