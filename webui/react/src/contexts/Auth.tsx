import { generateContext } from 'contexts';
import { globalStorage } from 'globalStorage';
import { updateDetApi } from 'services/apiConfig';
import { Auth } from 'types';

enum ActionType {
  MarkChecked,
  Reset,
  ResetChecked,
  Set,
}

export const AUTH_COOKIE_KEY = 'auth';
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
  | { type: ActionType.MarkChecked }
  | { type: ActionType.Reset }
  | { type: ActionType.ResetChecked }
  | { type: ActionType.Set; value: Auth }

const defaultAuth: State = {
  checked: false,
  isAuthenticated: false,
};

const clearAuthCookie = (): void => {
  document.cookie = `${AUTH_COOKIE_KEY}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case ActionType.MarkChecked:
      return { ...state, checked: true };
    case ActionType.Reset:
      clearAuthCookie();
      globalStorage.removeAuthToken();
      updateDetApi({ apiKey: undefined });
      return { ...defaultAuth };
    case ActionType.ResetChecked:
      return { ...state, checked: false };
    case ActionType.Set:
      if (action.value.token) {
        globalStorage.authToken = action.value.token;
        updateDetApi({ apiKey: 'Bearer ' + action.value.token });
      }
      return { ...action.value, checked: true };
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
