import axios, { CancelToken, CancelTokenSource } from 'axios';
import { useCallback, useEffect, useState } from 'react';

import Auth from 'contexts/Auth';
import handleError, { ErrorType } from 'ErrorHandler';
import { getCurrentUser } from 'services/api';
import { getCookie } from 'utils/browser';

const useAuthCheck = (): (() => void) => {
  const setAuth = Auth.useActionContext();
  const [ source, setSource ] = useState<CancelTokenSource | undefined>();

  const triggerCheckAuth = useCallback(() => {
    setSource(axios.CancelToken.source());
  }, []);

  useEffect(() => setAuth({ type: Auth.ActionType.ResetChecked }), [ setAuth ]);

  useEffect(() => {
    const checkAuth = async (cancelToken: CancelToken): Promise<void> => {
      const authCookie = getCookie('auth');
      if (!authCookie) {
        setAuth({ type: Auth.ActionType.MarkChecked });
        return;
      }

      try {
        const user = await getCurrentUser({ cancelToken });
        setAuth({ type: Auth.ActionType.Set, value: { isAuthenticated: true, user } });
      } catch (e) {
        handleError({
          error: e,
          isUserTriggered: false,
          message: e.message,
          publicMessage: 'Unable to verify current user.',
          publicSubject: 'GET user failed',
          silent: true,
          type: ErrorType.Auth,
        });
        setAuth({ type: Auth.ActionType.Reset });
        setAuth({ type: Auth.ActionType.MarkChecked });
      }
    };

    if (source) checkAuth(source.token);

    return source?.cancel;
  }, [ setAuth, source ]);

  return triggerCheckAuth;
};

export default useAuthCheck;
