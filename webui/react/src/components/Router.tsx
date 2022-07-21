import React, { useEffect, useState } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';

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
    <Routes>
      {props.routes.map((config) => {
        const { element, ...route } = config;

        if (route.needAuth && !auth.isAuthenticated) {
          // Do not mount login page until auth is checked.
          if (!auth.checked) return <Route key={route.id} {...route} element={element} />;
          return (
            <Route
              key={route.id}
              {...route}
              element={(
                <Navigate
                  state={{ loginRedirect: filterOutLoginLocation(location) }}
                  to={paths.login()}
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
            return (
              <Route
                element={<Navigate to={route.redirect} />}
                key={route.id}
                path={route.path}
              />
            );
          } else {
            return (
              <Route element={<Navigate to={route.redirect} />} key={route.id} path={route.path} />
            );
          }
        }
        return <Route element={element} key={route.id} {...route} />;
      })}
    </Routes>
  );
};

export default Router;
