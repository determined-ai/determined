import axios, { CancelToken } from 'axios';
import { useCallback, useEffect, useState } from 'react';

import Auth from 'contexts/Auth';
import handleError, { ErrorType } from 'ErrorHandler';
import { getCurrentUser } from 'services/api';

const useAuthCheck = (): [ () => void, number ] => {
  const setAuth = Auth.useActionContext();
  const [ triggerCount, setTriggerCount ] = useState(0);
  const [ source, setSource ] = useState(axios.CancelToken.source());

  const triggerCheckAuth = useCallback(() => {
    setSource(axios.CancelToken.source());
  }, []);

  useEffect(() => {
    const checkAuth = async (cancelToken: CancelToken): Promise<void> => {
      setTriggerCount(prev => prev + 1);

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
      }
    };

    checkAuth(source.token);

    return source.cancel;
  }, [ setAuth, source ]);

  return [ triggerCheckAuth, triggerCount ];
};

export default useAuthCheck;
