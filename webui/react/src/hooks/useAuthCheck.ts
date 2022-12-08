import queryString from 'query-string';
import { useCallback } from 'react';
import { useLocation } from 'react-router-dom';

import { AUTH_COOKIE_KEY, StoreAction, useStoreDispatch } from 'contexts/Store';
import { globalStorage } from 'globalStorage';
import { routeAll } from 'routes/utils';
import { getCurrentUser } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { ErrorType } from 'shared/utils/error';
import { isAborted, isAuthFailure } from 'shared/utils/service';
import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { getCookie } from 'utils/browser';
import handleError from 'utils/error';
import { Loadable } from 'utils/loadable';

const useAuthCheck = (canceler: AbortController): (() => void) => {
  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
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
  }, [info.externalLoginUri]);

  const checkAuth = useCallback(async (): Promise<void> => {
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

      try {
        const user = await getCurrentUser({ signal: canceler.signal });
        storeDispatch({
          type: StoreAction.SetAuth,
          value: { isAuthenticated: true, token: authToken, user },
        });
      } catch (e) {
        if (isAborted(e)) return;

        const isAuthError = isAuthFailure(e, !!info.externalLoginUri);
        handleError(e, {
          isUserTriggered: false,
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
      } finally {
        storeDispatch({ type: StoreAction.SetAuthCheck });
      }
    } else if (info.externalLoginUri) {
      redirectToExternalSignin();
    } else {
      storeDispatch({ type: StoreAction.SetAuthCheck });
    }
  }, [
    canceler,
    info.externalLoginUri,
    location.search,
    redirectToExternalSignin,
    storeDispatch,
    updateBearerToken,
  ]);

  return checkAuth;
};

export default useAuthCheck;
