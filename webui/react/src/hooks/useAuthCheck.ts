import queryString from 'query-string';
import { useCallback, useEffect } from 'react';
import { useHistory, useLocation } from 'react-router';

import { AUTH_COOKIE_KEY, StoreAction, useStore, useStoreDispatch } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import { globalStorage } from 'globalStorage';
import { paths, routeAll } from 'routes/utils';
import { getCurrentUser, isAuthFailure } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { isAborted } from 'services/utils';
import { getCookie } from 'utils/browser';

const useAuthCheck = (canceler: AbortController): (() => void) => {
  const { info } = useStore();
  const history = useHistory();
  const location = useLocation();
  const storeDispatch = useStoreDispatch();

  const updateBearerToken = useCallback((token: string) => {
    globalStorage.authToken = token;
    updateDetApi({ apiKey: `Bearer ${token}` });
  }, []);

  const redirectToExternalSignin = useCallback(() => {
    const redirect = encodeURIComponent(window.location.href);
    const authUrl = `${info.externalLoginUri}?redirect=${redirect}`;
    routeAll(authUrl);
  }, [ info.externalLoginUri ]);

  const checkAuth = useCallback(async (): Promise<void> => {
    const { jwt } = queryString.parse(location.search);
    /*
     * Check for an auth token in the cookie from SSO and
     * update the storage token and the api to use the cookie token.
     * If auth token is not found, look for `jwt` query param instead.
     */
    const jwtToken = jwt && !Array.isArray(jwt) ? jwt : null;
    const cookieToken = getCookie(AUTH_COOKIE_KEY);
    const authToken = jwtToken ?? cookieToken ?? globalStorage.authToken;
    updateBearerToken(authToken);

    try {
      const user = await getCurrentUser({ signal: canceler.signal });
      storeDispatch({
        type: StoreAction.SetAuth,
        value: { isAuthenticated: true, token: authToken, user },
      });
    } catch (e) {
      if (isAborted(e)) return;

      const isAuthError = isAuthFailure(e, !!info.externalLoginUri);
      handleError({
        error: e,
        isUserTriggered: false,
        message: e.message,
        publicMessage: 'Unable to verify current user.',
        publicSubject: 'GET user failed',
        silent: true,
        type: isAuthError ? ErrorType.Auth : ErrorType.Server,
      });

      if (isAuthError) {
        updateDetApi({ apiKey: undefined });
        storeDispatch({ type: StoreAction.ResetAuth });

        if (info.externalLoginUri) redirectToExternalSignin();
      }

      // Handle JWT failures with missing `externalLoginUri`.
      if (jwt) history.replace(paths.clusterNotAvailable());
    } finally {
      storeDispatch({ type: StoreAction.SetAuthCheck });
    }
  }, [
    canceler,
    history,
    info.externalLoginUri,
    location.search,
    redirectToExternalSignin,
    storeDispatch,
    updateBearerToken,
  ]);

  useEffect(() => storeDispatch({ type: StoreAction.ResetAuthCheck }), [ storeDispatch ]);

  return checkAuth;
};

export default useAuthCheck;
