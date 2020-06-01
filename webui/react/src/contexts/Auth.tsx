import { generateContext } from 'contexts';
import { Auth } from 'types';

enum ActionType {
  Reset,
  ResetCheck,
  Set,
  UpdateCheck,
}

/*
 * `checkCount` allows the `useAuthCheck` hook to keep tabs of how many times
 * is has been called in sign in. It is kept here to avoid a situation where
 * `isAuthenticated` is off sync with `checkCount`, which causes the Sign In
 * form to flicker briefly before being redirected to an authenticated page.
 */
type State = Auth & {
  checked: boolean;
};

type Action =
  | { type: ActionType.Reset }
  | { type: ActionType.ResetCheck }
  | { type: ActionType.Set; value: Auth }
  | { type: ActionType.UpdateCheck }

const defaultAuth: State = {
  checked: false,
  isAuthenticated: false,
};

const clearAuthCookie = (): void => {
  document.cookie = 'auth=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;';
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.Reset:
      clearAuthCookie();
      return { ...defaultAuth };
    case ActionType.ResetCheck:
      return { ...state, checked: false };
    case ActionType.Set:
      return { ...action.value, checked: true };
    case ActionType.UpdateCheck:
      return { ...state, checked: true };
    default:
      return state;
  }
};

const contextProvider = generateContext<State, Action>({
  initialState: defaultAuth,
  name: 'Auth',
  reducer,
});

export default { ...contextProvider, ActionType };
