import { CancelToken } from 'axios';
import { Dispatch } from 'react';

import { generateContext } from 'contexts';
import { getCurrentUser, isAuthFailure } from 'services/api';
import { Auth } from 'types';

enum ActionType {
  Reset,
  Set,
}

type State = Auth;

type Action =
  | { type: ActionType.Reset}
  | { type: ActionType.Set; value: Auth }

const defaultAuth: Auth = { isAuthenticated: false };

const clearAuthCookie = (): void => {
  document.cookie = 'auth=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.Reset:
      clearAuthCookie();
      return defaultAuth;
    case ActionType.Set:
      return action.value;
    default:
      return state;
  }
};

const contextProvider = generateContext<Auth, Action>({
  initialState: defaultAuth,
  name: 'Auth',
  reducer,
});

export const updateAuth =
  async (setAuth: Dispatch<Action>, cancelToken?: CancelToken): Promise<boolean> => {
    try{
      const user = await getCurrentUser({ cancelToken });
      setAuth({ type: ActionType.Set, value: { isAuthenticated: true, user } });
      return true;
    } catch (e) {
      // could use a retry mechanism on non-credential related failures
      if (isAuthFailure(e)) {
        setAuth({ type: ActionType.Reset });
      }
      return false;
    }
  };

export default { ...contextProvider, ActionType };
