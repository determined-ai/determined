import { generateContext } from 'contexts';
import { Auth } from 'types';

enum ActionType {
  Reset,
  Set,
}

type State = Auth;

type Action =
  | { type: ActionType.Reset }
  | { type: ActionType.Set; value: Auth }

const defaultAuth: Auth = { isAuthenticated: false };

const clearAuthCookie = (): void => {
  document.cookie = 'auth=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.Reset:
      clearAuthCookie();
      return { ...defaultAuth };
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

export default { ...contextProvider, ActionType };
