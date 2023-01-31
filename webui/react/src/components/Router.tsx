import React, { useEffect, useState } from 'react';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';

import useTheme from 'hooks/useTheme';
import { paths } from 'routes/utils';
import useUI from 'shared/contexts/stores/UI';
import { RouteConfig } from 'shared/types';
import { filterOutLoginLocation } from 'shared/utils/routes';
import { useAuth } from 'stores/auth';
import { Loadable } from 'utils/loadable';

interface Props {
  routes: RouteConfig[];
}

const Router: React.FC<Props> = (props: Props) => {
  const loadableAuth = useAuth();
  const isAuthenticated = Loadable.match(loadableAuth.auth, {
    Loaded: (auth) => auth.isAuthenticated,
    NotLoaded: () => false,
  });
  const authChecked = loadableAuth.authChecked;
  const [canceler] = useState(new AbortController());
  const { actions: uiActions } = useUI();
  const location = useLocation();

  useTheme();

  useEffect(() => {
    if (isAuthenticated) {
      uiActions.hideSpinner();
    }
  }, [isAuthenticated, uiActions]);

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  return (
    <Routes>
      {props.routes.map((config) => {
        const { element, ...route } = config;

        if (route.needAuth && !isAuthenticated) {
          // Do not mount login page until auth is checked.
          if (!authChecked) return <Route {...route} element={element} key={route.id} />;
          return (
            <Route
              {...route}
              element={<Navigate state={filterOutLoginLocation(location)} to={paths.login()} />}
              key={route.id}
            />
          );
        } else if (route.redirect) {
          /*
           * We treat '*' as a catch-all path and specifically avoid wrapping the
           * `Redirect` with a `DomRoute` component. This ensures the catch-all
           * redirect will occur when encountered in the `Switch` traversal.
           */
          if (route.path === '*') {
            return <Route element={<Navigate to={'/'} />} key={route.id} path={route.path} />;
          } else {
            return (
              <Route element={<Navigate to={route.redirect} />} key={route.id} path={route.path} />
            );
          }
        }

        return <Route {...route} element={element} key={route.id} path={route.path} />;
      })}
    </Routes>
  );
};

export default Router;
