import { generateContext } from 'contexts';
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
    case ActionType.SetIsAuthenticated:
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

export default { ...contextProvider, ActionType };
