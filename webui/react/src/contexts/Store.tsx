import React, { Dispatch, useContext, useReducer } from 'react';

import { StoreProvider as UIStoreProvider } from 'shared/contexts/stores/UI';
import { clone, isEqual } from 'shared/utils/data';
import rootLogger from 'shared/utils/Logger';
import { checkDeepEquality } from 'shared/utils/store';
import { UserAssignment, UserRole, Workspace } from 'types';

const logger = rootLogger.extend('store');

interface Props {
  children?: React.ReactNode;
}

interface OmnibarState {
  isShowing: boolean;
}

interface State {
  knownRoles: UserRole[];
  pinnedWorkspaces: Workspace[];

  ui: {
    omnibar: OmnibarState;
  };
  userAssignments: UserAssignment[];
}

export const StoreAction = {
  // Omnibar
  HideOmnibar: 'HideOmnibar',

  Reset: 'Reset',

  // User assignments, roles, and derived permissions
  SetKnownRoles: 'SetKnownRoles',

  // PinnedWorkspaces
  SetPinnedWorkspaces: 'SetPinnedWorkspaces',

  // User Settings
  SetUserSettings: 'SetUserSettings',
  ShowOmnibar: 'ShowOmnibar',
} as const;

type Action =
  | { type: typeof StoreAction.Reset }
  | { type: typeof StoreAction.SetPinnedWorkspaces; value: Workspace[] }
  | { type: typeof StoreAction.HideOmnibar }
  | { type: typeof StoreAction.ShowOmnibar }
  | { type: typeof StoreAction.SetKnownRoles; value: UserRole[] };

const initState: State = {
  knownRoles: [],
  pinnedWorkspaces: [],
  ui: { omnibar: { isShowing: false } }, // TODO move down a level
  userAssignments: [],
};

const StateContext = React.createContext<State | undefined>(undefined);
const DispatchContext = React.createContext<Dispatch<Action> | undefined>(undefined);

// TODO turn this into a partial reducer simliar to reducerUI.
const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case StoreAction.Reset:
      return clone(initState) as State;
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
