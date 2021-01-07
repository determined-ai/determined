import { useCallback, useEffect, useState } from 'react';

import Auth, { AUTH_COOKIE_KEY } from 'contexts/Auth';
import handleError, { ErrorType } from 'ErrorHandler';
import { globalStorage } from 'globalStorage';
import { getCurrentUser, isAuthFailure } from 'services/api';
import { isAborted } from 'services/utils';
import { getCookie } from 'utils/browser';

const useAuthCheck = (): (() => void) => {
  const setAuth = Auth.useActionContext();
  const [ canceler, setCanceler ] = useState<AbortController>();

  const triggerCheckAuth = useCallback(() => setCanceler(new AbortController()), []);

  useEffect(() => setAuth({ type: Auth.ActionType.ResetChecked }), [ setAuth ]);

  useEffect(() => {
    const checkAuth = async (signal: AbortSignal): Promise<void> => {
      const authToken = getCookie(AUTH_COOKIE_KEY) || globalStorage.authToken;
      if (!authToken) {
        setAuth({ type: Auth.ActionType.MarkChecked });
        return;
      }

      try {
        const user = await getCurrentUser({ signal });
        setAuth({
          type: Auth.ActionType.Set,
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
          setAuth({ type: Auth.ActionType.Reset });
          setAuth({ type: Auth.ActionType.MarkChecked });
        }
      }
    };

    if (canceler) checkAuth(canceler.signal);

    return () => canceler?.abort();
  }, [ canceler, setAuth ]);

  return triggerCheckAuth;
};

export default useAuthCheck;
