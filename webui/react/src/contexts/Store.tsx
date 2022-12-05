import React, { Dispatch, useContext, useReducer } from 'react';

import { globalStorage } from 'globalStorage';
import { StoreProvider as UIStoreProvider } from 'shared/contexts/stores/UI';
import { clone, isEqual } from 'shared/utils/data';
import rootLogger from 'shared/utils/Logger';
import { checkDeepEquality } from 'shared/utils/store';
import { Auth, DetailedUser, UserAssignment, UserRole, Workspace } from 'types';
import { getCookie, setCookie } from 'utils/browser';

const logger = rootLogger.extend('store');

interface Props {
  children?: React.ReactNode;
}

interface OmnibarState {
  isShowing: boolean;
}

interface State {
  auth: Auth & { checked: boolean };

  knownRoles: UserRole[];
  pinnedWorkspaces: Workspace[];

  ui: {
    omnibar: OmnibarState;
  };
  userAssignments: UserAssignment[];
  userRoles: UserRole[];
  users: DetailedUser[];
}

export const StoreAction = {
  // Omnibar
  HideOmnibar: 'HideOmnibar',

  Reset: 'Reset',
  // Auth
  ResetAuth: 'ResetAuth',

  ResetAuthCheck: 'ResetAuthCheck',

  // Agents
  SetAgents: 'SetAgents',

  SetAuth: 'SetAuth',

  SetAuthCheck: 'SetAuthCheck',

  SetCurrentUser: 'SetCurrentUser',

  // User assignments, roles, and derived permissions
  SetKnownRoles: 'SetKnownRoles',

  // PinnedWorkspaces
  SetPinnedWorkspaces: 'SetPinnedWorkspaces',

  SetUserAssignments: 'SetUserAssignments',

  SetUserRoles: 'SetUserRoles',

  // Users
  SetUsers: 'SetUsers',
  // User Settings
  SetUserSettings: 'SetUserSettings',
  ShowOmnibar: 'ShowOmnibar',
} as const;

type Action =
  | { type: typeof StoreAction.Reset }
  | { type: typeof StoreAction.ResetAuth }
  | { type: typeof StoreAction.ResetAuthCheck }
  | { type: typeof StoreAction.SetAuth; value: Auth }
  | { type: typeof StoreAction.SetAuthCheck }
  | { type: typeof StoreAction.SetUsers; value: DetailedUser[] }
  | { type: typeof StoreAction.SetCurrentUser; value: DetailedUser }
  | { type: typeof StoreAction.SetPinnedWorkspaces; value: Workspace[] }
  | { type: typeof StoreAction.HideOmnibar }
  | { type: typeof StoreAction.ShowOmnibar }
  | { type: typeof StoreAction.SetKnownRoles; value: UserRole[] }
  | { type: typeof StoreAction.SetUserRoles; value: UserRole[] }
  | { type: typeof StoreAction.SetUserAssignments; value: UserAssignment[] };

export const AUTH_COOKIE_KEY = 'auth';

const initAuth = {
  checked: false,
  isAuthenticated: false,
};

const initState: State = {
  auth: initAuth,
  knownRoles: [],
  pinnedWorkspaces: [],
  ui: { omnibar: { isShowing: false } }, // TODO move down a level
  userAssignments: [],
  userRoles: [
    {
      id: -10,
      name: 'INITIALIZATION',
      permissions: [],
    },
  ],
  users: [],
};

const StateContext = React.createContext<State | undefined>(undefined);
const DispatchContext = React.createContext<Dispatch<Action> | undefined>(undefined);

const clearAuthCookie = (): void => {
  document.cookie = `${AUTH_COOKIE_KEY}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
};

/**
 * set the auth cookie if it's not already set.
 * @param token auth token
 */
const ensureAuthCookieSet = (token: string): void => {
  if (!getCookie(AUTH_COOKIE_KEY)) setCookie(AUTH_COOKIE_KEY, token);
};

// TODO turn this into a partial reducer simliar to reducerUI.
const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case StoreAction.Reset:
      return clone(initState) as State;
    case StoreAction.ResetAuth:
      clearAuthCookie();
      globalStorage.removeAuthToken();
      return { ...state, auth: { ...state.auth, isAuthenticated: initAuth.isAuthenticated } };
    case StoreAction.ResetAuthCheck:
      if (!state.auth.checked) return state;
      return { ...state, auth: { ...state.auth, checked: false } };
    case StoreAction.SetAuth:
      if (action.value.token) {
        /**
         * project Samuel provisioned auth doesn't set a cookie
         * like our other auth methods do.
         *
         */
        ensureAuthCookieSet(action.value.token);
        globalStorage.authToken = action.value.token;
      }
      return { ...state, auth: { ...action.value, checked: true } };
    case StoreAction.SetAuthCheck:
      if (state.auth.checked) return state;
      return { ...state, auth: { ...state.auth, checked: true } };
    case StoreAction.SetUsers:
      if (isEqual(state.users, action.value)) return state;
      return { ...state, users: action.value };
    case StoreAction.SetCurrentUser: {
      if (isEqual(action.value, state.auth.user)) return state;
      const users = [...state.users];
      const userIdx = users.findIndex((user) => user.id === action.value.id);
      if (userIdx > -1) users[userIdx] = { ...users[userIdx], ...action.value };
      return { ...state, auth: { ...state.auth, user: action.value }, users };
    }
    case StoreAction.SetPinnedWorkspaces:
      if (isEqual(state.pinnedWorkspaces, action.value)) return state;
      return { ...state, pinnedWorkspaces: action.value };
    case StoreAction.HideOmnibar:
      if (!state.ui.omnibar.isShowing) return state;
      return { ...state, ui: { ...state.ui, omnibar: { ...state.ui.omnibar, isShowing: false } } };
    case StoreAction.ShowOmnibar:
      if (state.ui.omnibar.isShowing) return state;
      return { ...state, ui: { ...state.ui, omnibar: { ...state.ui.omnibar, isShowing: true } } };
    case StoreAction.SetKnownRoles:
      if (isEqual(state.knownRoles, action.value)) return state;
      return { ...state, knownRoles: action.value };
    case StoreAction.SetUserRoles:
      if (isEqual(state.userRoles, action.value)) return state;
      return { ...state, userRoles: action.value };
    case StoreAction.SetUserAssignments:
      if (isEqual(state.userAssignments, action.value)) return state;
      return { ...state, userAssignments: action.value };
    default:
      return state;
  }
};

export const useStore = (): State => {
  const context = useContext(StateContext);
  if (context === undefined) {
    throw new Error('useStore must be used within a StoreProvider');
  }
  return context;
};

export const useStoreDispatch = (): Dispatch<Action> => {
  const context = useContext(DispatchContext);
  if (context === undefined) {
    throw new Error('useStoreDispatch must be used within a StoreProvider');
  }
  return context;
};

const StoreProvider: React.FC<Props> = ({ children }: Props) => {
  const [state, dispatch] = useReducer(checkDeepEquality(reducer, logger), initState);
  return (
    <StateContext.Provider value={state}>
      <DispatchContext.Provider value={dispatch}>{children}</DispatchContext.Provider>
    </StateContext.Provider>
  );
};

/** a set of app level store providers */
const StackedStoreProvider: React.FC<Props> = ({ children }: Props) => {
  return (
    <StoreProvider>
      <UIStoreProvider>{children}</UIStoreProvider>
    </StoreProvider>
  );
};

export default StackedStoreProvider;
