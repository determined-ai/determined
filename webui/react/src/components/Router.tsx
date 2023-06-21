import { useObservable } from 'micro-observables';
import React, { useEffect } from 'react';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';

import { paths } from 'routes/utils';
import authStore from 'stores/auth';
import useUI from 'stores/contexts/UI';
import { RouteConfig } from 'types';
import { filterOutLoginLocation } from 'utils/routes';

interface Props {
  routes: RouteConfig[];
}

const Router: React.FC<Props> = (props: Props) => {
  const isAuthenticated = useObservable(authStore.isAuthenticated);
  const { actions: uiActions } = useUI();
  const location = useLocation();

  useEffect(() => {
    if (isAuthenticated) {
      uiActions.hideSpinner();
    }
  }, [isAuthenticated, uiActions]);

  return (
    <Routes>
      {props.routes.map((config) => {
        const { element, ...route } = config;

        if (route.needAuth && !isAuthenticated) {
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
