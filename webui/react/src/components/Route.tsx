import axios, { CancelToken } from 'axios';
import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Route as DomRoute, Redirect } from 'react-router-dom';

import Spinner from 'components/Spinner';
import Auth from 'contexts/Auth';
import { crossoverRoute, RouteConfigItem } from 'routes';
import { getCurrentUser } from 'services/api';

/*
 * A wrapper for <Route> that redirects to the login
 * screen if you're not yet authenticated.
 */

const Route: React.FC<RouteConfigItem> = (props: RouteConfigItem) => {
  const mounted = useRef(false);
  const auth = Auth.useStateContext();
  const setAuth = Auth.useActionContext();
  const needAuth = props.needAuth;

  // isLoading is true at the start until useEffect overrides it.
  const [ isLoading, setIsLoading ] = useState(true);

  const checkAuth = useCallback(async (cancelToken: CancelToken): Promise<void> => {
    try {
      const user = await getCurrentUser(cancelToken);

      if (mounted.current) {
        setAuth({
          type: Auth.ActionType.Set,
          value: { isAuthenticated: true, user },
        });
      }
    } catch (e) {
      // TODO: Update to internal routing when React takes over login.
      crossoverRoute('/ui/logout');
    }
  }, [ setAuth ]);

  const setLoading = (loadingStatus: boolean): void => {
    if (mounted.current) setIsLoading(loadingStatus);
  };

  useEffect(() => {
    const source = axios.CancelToken.source();
    // Keeps track of whether component has mounted or not
    mounted.current = true;

    // Use IIFE to make block sync
    (async (): Promise<void> => {
      if (needAuth) await checkAuth(source.token);
      setLoading(false);
    })();

    /*
     * Return cancellable function to interrupt fetchUser if we are
     * removing this component before the HTTP call completes.
     */
    return (): void => {
      mounted.current = false;
      source.cancel();
    };
  }, [ checkAuth, needAuth ]);

  if (isLoading) return <Spinner fullPage={true} />;
  if (needAuth && !auth.isAuthenticated) return <Redirect to="/ui/login" />;
  return <DomRoute {...props} />;
};

export default Route;
