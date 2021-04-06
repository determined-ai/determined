import React, { Dispatch, useContext, useReducer } from 'react';

import { globalStorage } from 'globalStorage';
import { Agent, Auth, ClusterOverview, DetailedUser, DeterminedInfo, ResourceType } from 'types';
import { updateFaviconType } from 'utils/browser';
import { clone, isEqual } from 'utils/data';
import { percent } from 'utils/number';

interface Props {
  children?: React.ReactNode;
}

interface UI {
  chromeCollapsed: boolean;
  showChrome: boolean;
  showSpinner: boolean;
}

interface State {
  agents: Agent[];
  auth: Auth & { checked: boolean };
  cluster: ClusterOverview;
  info: DeterminedInfo;
  ui: UI;
  users: DetailedUser[];
}

export enum StoreActionType {
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

  // UI
  CollapseUIChrome,
  ExpandUIChrome,
  HideUIChrome,
  HideUISpinner,
  ShowUIChrome,
  ShowUISpinner,

  // Users
  SetUsers,
}

type Action =
| { type: StoreActionType.Reset }
| { type: StoreActionType.SetAgents; value: Agent[] }
| { type: StoreActionType.ResetAuth }
| { type: StoreActionType.ResetAuthCheck }
| { type: StoreActionType.SetAuth; value: Auth }
| { type: StoreActionType.SetAuthCheck }
| { type: StoreActionType.SetInfo; value: DeterminedInfo }
| { type: StoreActionType.CollapseUIChrome }
| { type: StoreActionType.ExpandUIChrome }
| { type: StoreActionType.HideUIChrome }
| { type: StoreActionType.HideUISpinner }
| { type: StoreActionType.ShowUIChrome }
| { type: StoreActionType.ShowUISpinner }
| { type: StoreActionType.SetUsers; value: DetailedUser[] }

export const AUTH_COOKIE_KEY = 'auth';

const initAuth = {
  checked: false,
  isAuthenticated: false,
};
const initResourceTally = { allocation:0, available: 0, total: 0 };
const initClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: clone(initResourceTally),
  [ResourceType.GPU]: clone(initResourceTally),
  [ResourceType.ALL]: clone(initResourceTally),
  [ResourceType.UNSPECIFIED]: clone(initResourceTally),
};
const initInfo = {
  clusterId: '',
  clusterName: '',
  isTelemetryEnabled: false,
  masterId: '',
  version: process.env.VERSION || '',
};
const initUI = {
  chromeCollapsed: false,
  showChrome: true,
  showSpinner: false,
};
const initState: State = {
  agents: [],
  auth: initAuth,
  cluster: initClusterOverview,
  info: initInfo,
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

const reducer = (state: State, action: Action) => {
  switch (action.type) {
    case StoreActionType.Reset:
      return clone(initState);
    case StoreActionType.SetAgents: {
      if (action.value.length === 0) return state;
      if (isEqual(state.agents, action.value)) return state;
      const cluster = agentsToOverview(action.value);
      updateFaviconType(cluster[ResourceType.ALL].allocation !== 0);
      return { ...state, agents: action.value, cluster };
    }
    case StoreActionType.ResetAuth:
      clearAuthCookie();
      globalStorage.removeAuthToken();
      return { ...state, auth: { ...initAuth } };
    case StoreActionType.ResetAuthCheck:
      if (!state.auth.checked) return state;
      return { ...state, auth: { ...state.auth, checked: false } };
    case StoreActionType.SetAuth:
      if (action.value.token) {
        globalStorage.authToken = action.value.token;
      }
      return { ...state, auth: { ...action.value, checked: true } };
    case StoreActionType.SetAuthCheck:
      if (state.auth.checked) return state;
      return { ...state, auth: { ...state.auth, checked: true } };
    case StoreActionType.SetInfo:
      if (isEqual(state.info, action.value)) return state;
      return { ...state, info: action.value };
    case StoreActionType.CollapseUIChrome:
      if (state.ui.chromeCollapsed) return state;
      return { ...state, ui: { ...state.ui, chromeCollapsed: true } };
    case StoreActionType.ExpandUIChrome:
      if (!state.ui.chromeCollapsed) return state;
      return { ...state, ui: { ...state.ui, chromeCollapsed: false } };
    case StoreActionType.HideUIChrome:
      if (!state.ui.showChrome) return state;
      return { ...state, ui: { ...state.ui, showChrome: false } };
    case StoreActionType.HideUISpinner:
      if (!state.ui.showSpinner) return state;
      return { ...state, ui: { ...state.ui, showSpinner: false } };
    case StoreActionType.ShowUIChrome:
      if (state.ui.showChrome) return state;
      return { ...state, ui: { ...state.ui, showChrome: true } };
    case StoreActionType.ShowUISpinner:
      if (state.ui.showSpinner) return state;
      return { ...state, ui: { ...state.ui, showSpinner: true } };
    case StoreActionType.SetUsers:
      if (isEqual(state.users, action.value)) return state;
      return { ...state, users: action.value };
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
