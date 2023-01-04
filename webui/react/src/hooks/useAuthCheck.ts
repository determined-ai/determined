import queryString from 'query-string';
import { useCallback } from 'react';
import { useLocation } from 'react-router-dom';

import { globalStorage } from 'globalStorage';
import { routeAll } from 'routes/utils';
import { updateDetApi } from 'services/apiConfig';
import { AUTH_COOKIE_KEY, useAuth } from 'stores/auth';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { getCookie } from 'utils/browser';
import { Loadable } from 'utils/loadable';

const useAuthCheck = (): (() => void) => {
  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
  const location = useLocation();
  const { setAuth, setAuthCheck } = useAuth();

  const updateBearerToken = useCallback((token: string) => {
    globalStorage.authToken = token;
    updateDetApi({ apiKey: `Bearer ${token}` });
  }, []);

  const redirectToExternalSignin = useCallback(() => {
    const redirect = encodeURIComponent(window.location.href);
    const authUrl = `${info.externalLoginUri}?redirect=${redirect}`;
    routeAll(authUrl);
  }, [info.externalLoginUri]);

  const checkAuth = useCallback((): void => {
    /*
     * Check for the auth token from the following sources:
     *   1 - query param jwt from external authentication.
     *   2 - server cookie
     *   3 - local storage
     */
    const { jwt } = queryString.parse(location.search);
    const jwtToken = jwt && !Array.isArray(jwt) ? jwt : null;
    const cookieToken = getCookie(AUTH_COOKIE_KEY);
    const authToken = jwtToken ?? cookieToken ?? globalStorage.authToken;

    /*
     * If auth token found, update the API bearer token and validate it with the current user API.
     * If an external login URL is provided, redirect there.
     * Otherwise mark that we checked the auth and skip auth token validation.
     */

    if (authToken) {
      updateBearerToken(authToken);

      setAuth({ isAuthenticated: true, token: authToken });
      setAuthCheck();
    } else if (info.externalLoginUri) {
      redirectToExternalSignin();
    } else {
      setAuthCheck();
    }
  }, [
    info.externalLoginUri,
    location.search,
    setAuth,
    setAuthCheck,
    redirectToExternalSignin,
    updateBearerToken,
  ]);

  return checkAuth;
};

export default useAuthCheck;
