import React, { Dispatch, useContext, useReducer } from 'react';

import { globalStorage } from 'globalStorage';
import {
  Agent, Auth, BrandingType, ClusterOverview, ClusterOverviewResource,
  DetailedUser, DeterminedInfo, ResourcePool, ResourceType,
} from 'types';
import { clone, isEqual } from 'utils/data';
import { percent } from 'utils/number';

interface Props {
  children?: React.ReactNode;
}

interface OmnibarState {
  isShowing: boolean;
}

interface UI {
  chromeCollapsed: boolean;
  omnibar: OmnibarState;
  showChrome: boolean;
  showSpinner: boolean;
}

export interface State {
  agents: Agent[];
  auth: Auth & { checked: boolean };
  cluster: ClusterOverview;
  info: DeterminedInfo;
  resourcePools: ResourcePool[];
  ui: UI;
  users: DetailedUser[];
}

export enum StoreAction {
  Reset,

  // Agents
  SetAgents,

  // Auth
  ResetAuth,
  ResetAuthCheck,
  SetAuth,
  SetAuthCheck,

  // Info
  SetInfo,
  SetInfoCheck,

  // UI
  HideUIChrome,
  HideUISpinner,
  ShowUIChrome,
  ShowUISpinner,

  // Users
  SetUsers,

  // Omnibar
  HideOmnibar,
  ShowOmnibar,

  // ResourcePools
  SetResourcePools,
}

export type Action =
| { type: StoreAction.Reset }
| { type: StoreAction.SetAgents; value: Agent[] }
| { type: StoreAction.ResetAuth }
| { type: StoreAction.ResetAuthCheck }
| { type: StoreAction.SetAuth; value: Auth }
| { type: StoreAction.SetAuthCheck }
| { type: StoreAction.SetInfo; value: DeterminedInfo }
| { type: StoreAction.SetInfoCheck }
| { type: StoreAction.HideUIChrome }
| { type: StoreAction.HideUISpinner }
| { type: StoreAction.ShowUIChrome }
| { type: StoreAction.ShowUISpinner }
| { type: StoreAction.SetUsers; value: DetailedUser[] }
| { type: StoreAction.SetResourcePools; value: ResourcePool[] }
| { type: StoreAction.HideOmnibar }
| { type: StoreAction.ShowOmnibar }

export const AUTH_COOKIE_KEY = 'auth';

const initAuth = {
  checked: false,
  isAuthenticated: false,
};
export const initResourceTally: ClusterOverviewResource = { allocation: 0, available: 0, total: 0 };
const initClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: clone(initResourceTally),
  [ResourceType.GPU]: clone(initResourceTally),
  [ResourceType.ALL]: clone(initResourceTally),
  [ResourceType.UNSPECIFIED]: clone(initResourceTally),
};
const initInfo = {
  branding: BrandingType.Determined,
  checked: false,
  clusterId: '',
  clusterName: '',
  isTelemetryEnabled: false,
  masterId: '',
  version: process.env.VERSION || '',
};
const initUI = {
  chromeCollapsed: false,
  omnibar: { isShowing: false },
  showChrome: true,
  showSpinner: false,
};
const initState: State = {
  agents: [],
  auth: initAuth,
  cluster: initClusterOverview,
  info: initInfo,
  resourcePools: [],
  ui: initUI,
  users: [],
};

const StateContext = React.createContext<State | undefined>(undefined);
const DispatchContext = React.createContext<Dispatch<Action> | undefined>(undefined);

const clearAuthCookie = (): void => {
  document.cookie = `${AUTH_COOKIE_KEY}=; expires=Thu, 01 Jan 1970 00:00:00 UTC; path=/;`;
};

export const agentsToOverview = (agents: Agent[]): ClusterOverview => {
  // Deep clone for render detection.
  const overview: ClusterOverview = clone(initClusterOverview);

  agents.forEach(agent => {
    agent.resources
      .filter(resource => resource.enabled)
      .forEach(resource => {
        const isResourceFree = resource.container == null;
        const availableResource = isResourceFree ? 1 : 0;
        overview[resource.type].available += availableResource;
        overview[resource.type].total++;
        overview[ResourceType.ALL].available += availableResource;
        overview[ResourceType.ALL].total++;
      });
  });

  for (const key in overview) {
    const rt = key as ResourceType;
    overview[rt].allocation = overview[rt].total !== 0 ?
      percent((overview[rt].total - overview[rt].available) / overview[rt].total) : 0;
  }

  return overview;
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case StoreAction.Reset:
      return clone(initState) as State;
    case StoreAction.SetAgents: {
      if (isEqual(state.agents, action.value)) return state;
      const cluster = agentsToOverview(action.value);
      return { ...state, agents: action.value, cluster };
    }
    case StoreAction.ResetAuth:
      clearAuthCookie();
      globalStorage.removeAuthToken();
      return { ...state, auth: { ...initAuth } };
    case StoreAction.ResetAuthCheck:
      if (!state.auth.checked) return state;
      return { ...state, auth: { ...state.auth, checked: false } };
    case StoreAction.SetAuth:
      if (action.value.token) {
        globalStorage.authToken = action.value.token;
      }
      return { ...state, auth: { ...action.value, checked: true } };
    case StoreAction.SetAuthCheck:
      if (state.auth.checked) return state;
      return { ...state, auth: { ...state.auth, checked: true } };
    case StoreAction.SetInfo:
      if (isEqual(state.info, action.value)) return state;
      return { ...state, info: action.value };
    case StoreAction.SetInfoCheck:
      return { ...state, info: { ...state.info, checked: true } };
    case StoreAction.HideUIChrome:
      if (!state.ui.showChrome) return state;
      return { ...state, ui: { ...state.ui, showChrome: false } };
    case StoreAction.HideUISpinner:
      if (!state.ui.showSpinner) return state;
      return { ...state, ui: { ...state.ui, showSpinner: false } };
    case StoreAction.ShowUIChrome:
      if (state.ui.showChrome) return state;
      return { ...state, ui: { ...state.ui, showChrome: true } };
    case StoreAction.ShowUISpinner:
      if (state.ui.showSpinner) return state;
      return { ...state, ui: { ...state.ui, showSpinner: true } };
    case StoreAction.SetUsers:
      if (isEqual(state.users, action.value)) return state;
      return { ...state, users: action.value };
    case StoreAction.SetResourcePools:
      if (isEqual(state.resourcePools, action.value)) return state;
      return { ...state, resourcePools: action.value };
    case StoreAction.HideOmnibar:
      if (!state.ui.omnibar.isShowing) return state;
      return { ...state, ui: { ...state.ui, omnibar: { ...state.ui.omnibar, isShowing: false } } };
    case StoreAction.ShowOmnibar:
      if (state.ui.omnibar.isShowing) return state;
      return { ...state, ui: { ...state.ui, omnibar: { ...state.ui.omnibar, isShowing: true } } };
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
  const [ state, dispatch ] = useReducer(reducer, initState);
  return (
    <StateContext.Provider value={state}>
      <DispatchContext.Provider value={dispatch}>
        {children}
      </DispatchContext.Provider>
    </StateContext.Provider>
  );
};

export default StoreProvider;
