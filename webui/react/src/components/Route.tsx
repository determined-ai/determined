import axios from 'axios';
import React, { useEffect, useRef, useState } from 'react';
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
  const source = axios.CancelToken.source();

  // isLoading is true at the start until useEffect overrides it.
  const [ isLoading, setIsLoading ] = useState(true);

  const checkAuth = async (): Promise<void> => {
    try {
      const user = await getCurrentUser(source.token);

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
  };

  const setLoading = (loadingStatus: boolean): void => {
    if (mounted.current) setIsLoading(loadingStatus);
  };

  useEffect(() => {
    // Keeps track of whether component has mounted or not
    mounted.current = true;

    // Use IIFE to make block sync
    (async (): Promise<void> => {
      if (props.needAuth) await checkAuth();
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
  }, []);

  if (isLoading) return <Spinner fullPage={true} />;
  if (props.needAuth && !auth.isAuthenticated) return <Redirect to="/ui/login" />;
  return <DomRoute {...props} />;
};

export default Route;
