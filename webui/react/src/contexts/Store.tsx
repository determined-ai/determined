import React, { Dispatch, useContext, useReducer } from 'react';

import { StoreProvider as UIStoreProvider } from 'shared/contexts/stores/UI';
import { clone, isEqual } from 'shared/utils/data';
import rootLogger from 'shared/utils/Logger';
import { checkDeepEquality } from 'shared/utils/store';
import { DeterminedInfo, UserAssignment, UserRole, Workspace } from 'types';

const logger = rootLogger.extend('store');

interface Props {
  children?: React.ReactNode;
}

interface OmnibarState {
  isShowing: boolean;
}

interface State {

  info: DeterminedInfo;
  knownRoles: UserRole[];
  pinnedWorkspaces: Workspace[];

  ui: {
    omnibar: OmnibarState;
  };
  userAssignments: UserAssignment[];
  userRoles: UserRole[];
}

export const StoreAction = {
  // Omnibar
  HideOmnibar: 'HideOmnibar',

  Reset: 'Reset',

  // Agents
  SetAgents: 'SetAgents',

  // Info
  SetInfo: 'SetInfo',

  SetInfoCheck: 'SetInfoCheck',

  // User assignments, roles, and derived permissions
  SetKnownRoles: 'SetKnownRoles',

  // PinnedWorkspaces
  SetPinnedWorkspaces: 'SetPinnedWorkspaces',

  SetUserAssignments: 'SetUserAssignments',

  SetUserRoles: 'SetUserRoles',

  // User Settings
  SetUserSettings: 'SetUserSettings',
  ShowOmnibar: 'ShowOmnibar',
} as const;

type Action =
  | { type: typeof StoreAction.Reset }
  | { type: typeof StoreAction.SetInfo; value: DeterminedInfo }
  | { type: typeof StoreAction.SetInfoCheck }
  | { type: typeof StoreAction.SetPinnedWorkspaces; value: Workspace[] }
  | { type: typeof StoreAction.HideOmnibar }
  | { type: typeof StoreAction.ShowOmnibar }
  | { type: typeof StoreAction.SetKnownRoles; value: UserRole[] }
  | { type: typeof StoreAction.SetUserRoles; value: UserRole[] }
  | { type: typeof StoreAction.SetUserAssignments; value: UserAssignment[] };

export const initInfo: DeterminedInfo = {
  branding: undefined,
  checked: false,
  clusterId: '',
  clusterName: '',
  featureSwitches: [],
  isTelemetryEnabled: false,
  masterId: '',
  rbacEnabled: false,
  version: process.env.VERSION || '',
};

const initState: State = {
  info: initInfo,
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
};

const StateContext = React.createContext<State | undefined>(undefined);
const DispatchContext = React.createContext<Dispatch<Action> | undefined>(undefined);

// TODO turn this into a partial reducer simliar to reducerUI.
const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case StoreAction.Reset:
      return clone(initState) as State;
    case StoreAction.SetInfo:
      if (isEqual(state.info, action.value)) return state;
      return { ...state, info: action.value };
    case StoreAction.SetInfoCheck:
      return { ...state, info: { ...state.info, checked: true } };
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
