import { CancelToken } from 'axios';
import { Dispatch } from 'react';

import { generateContext } from 'contexts';
import { getCurrentUser } from 'services/api';
import { Auth, User } from 'types';

enum ActionType {
  Reset,
  Set,
  SetUser,
  SetIsAuthenticated,
}

type State = Auth;

type Action =
  | { type: ActionType.Reset}
  | { type: ActionType.Set; value: Auth }
  | { type: ActionType.SetUser; value: User }
  | { type: ActionType.SetIsAuthenticated; value: boolean }

const defaultAuth: Auth = { isAuthenticated: false };

const clearAuthCookie = (): void => {
  // FIXME preserve other cookies?
  document.cookie = 'auth=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.Reset:
      clearAuthCookie();
      return defaultAuth;
    case ActionType.Set:
      return action.value;
    case ActionType.SetUser:
      return { ...state, user: action.value };
    case ActionType.SetIsAuthenticated: // DISCUSS are setUser and setisAuthenticated shortcuts? error prone
      if (!action.value) clearAuthCookie();
      return { ...state, isAuthenticated: action.value };
    default:
      return state;
  }
};

const contextProvider = generateContext<Auth, Action>({
  initialState: defaultAuth,
  name: 'Auth',
  reducer,
});

export const updateAuth = async (setAuth: Dispatch<Action>, cancelToken?: CancelToken): Promise<boolean> => {
  try{
    const user = await getCurrentUser({ cancelToken });
    setAuth({ type: ActionType.Set, value: { isAuthenticated: true, user } });
    return true;
  } catch (e) {
    // TODO check that it's an auth error otherwise throw an error
    setAuth({ type: ActionType.Reset });
    return false;
  }
};

export default { ...contextProvider, ActionType };
