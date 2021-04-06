import Auth, { AUTH_COOKIE_KEY } from 'contexts/Auth';
import { useCallback, useEffect } from 'react';

import { AUTH_COOKIE_KEY, StoreActionType, useStoreDispatch } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import { globalStorage } from 'globalStorage';
import { getCurrentUser, isAuthFailure } from 'services/api';
import { updateDetApi } from 'services/apiConfig';
import { isAborted } from 'services/utils';
import { getCookie } from 'utils/browser';

const useAuthCheck = (canceler: AbortController): (() => void) => {
  const storeDispatch = useStoreDispatch();

  const checkAuth = useCallback(async (): Promise<void> => {
    /*
     * Check for an auth token in the cookie from SSO and
     * update the storage token and the api to use the cookie token.
     */
    const cookieToken = getCookie(AUTH_COOKIE_KEY);
    if (cookieToken) {
      globalStorage.authToken = cookieToken;
      updateDetApi({ apiKey: `Bearer ${cookieToken}` });
    }

    /*
     * If a cookie token is not found, use the storage token if applicable.
     * Proceed to verify user only if there is an auth token.
     */
    const authToken = globalStorage.authToken;
    if (!authToken) {
      storeDispatch({ type: StoreActionType.SetAuthCheck });
      return;
    }

    try {
      const user = await getCurrentUser({ signal: canceler.signal });
      updateDetApi({ apiKey: `Bearer ${authToken}` });
      storeDispatch({
        type: StoreActionType.SetAuth,
        value: { isAuthenticated: true, token: authToken, user },
      });
    } catch (e) {
      if (isAborted(e)) return;
      const isAuthError = isAuthFailure(e);
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
        storeDispatch({ type: StoreActionType.ResetAuth });
        storeDispatch({ type: StoreActionType.SetAuthCheck });
      }
    }
  }, [ canceler, storeDispatch ]);

  useEffect(() => storeDispatch({ type: StoreActionType.ResetAuthCheck }), [ storeDispatch ]);

  return checkAuth;
};

export default useAuthCheck;
