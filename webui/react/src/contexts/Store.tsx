import React, { Dispatch, useContext, useReducer } from 'react';

import { globalStorage } from 'globalStorage';
import { V1UserWebSetting } from 'services/api-ts-sdk';
import { StoreProvider as UIStoreProvider } from 'shared/contexts/stores/UI';
import { clone, isEqual } from 'shared/utils/data';
import rootLogger from 'shared/utils/Logger';
import { percent } from 'shared/utils/number';
import { checkDeepEquality } from 'shared/utils/store';
import {
  Agent,
  Auth,
  ClusterOverview,
  ClusterOverviewResource,
  DetailedUser,
  DeterminedInfo,
  PoolOverview,
  ResourcePool,
  ResourceType,
  UserAssignment,
  UserRole,
  Workspace,
} from 'types';
import { getCookie, setCookie } from 'utils/browser';

const logger = rootLogger.extend('store');

interface Props {
  children?: React.ReactNode;
}

interface OmnibarState {
  isShowing: boolean;
}

interface State {
  activeExperiments: number;
  activeTasks: {
    commands: number;
    notebooks: number;
    shells: number;
    tensorboards: number;
  };
  agents: Agent[];
  auth: Auth & { checked: boolean };
  cluster: ClusterOverview;
  info: DeterminedInfo;
  knownRoles: UserRole[];
  pinnedWorkspaces: Workspace[];
  pool: PoolOverview;
  resourcePools: ResourcePool[];
  ui: {
    omnibar: OmnibarState;
  };
  userAssignments: UserAssignment[];
  userRoles: UserRole[];
  userSettings: V1UserWebSetting[];
  users: DetailedUser[];
}

export const StoreAction = {
  // Omnibar
  HideOmnibar: 'HideOmnibar',

  Reset: 'Reset',

  // Auth
  ResetAuth: 'ResetAuth',

  ResetAuthCheck: 'ResetAuthCheck',

  // Active Experiments
  SetActiveExperiments: 'SetActiveExperiments',

  // Tasks
  SetActiveTasks: 'SetActiveTasks',

  // Agents
  SetAgents: 'SetAgents',

  SetAuth: 'SetAuth',

  SetAuthCheck: 'SetAuthCheck',

  SetCurrentUser: 'SetCurrentUser',

  // Info
  SetInfo: 'SetInfo',

  SetInfoCheck: 'SetInfoCheck',

  // User assignments, roles, and derived permissions
  SetKnownRoles: 'SetKnownRoles',

  // PinnedWorkspaces
  SetPinnedWorkspaces: 'SetPinnedWorkspaces',

  // ResourcePools
  SetResourcePools: 'SetResourcePools',

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
  | { type: typeof StoreAction.SetAgents; value: Agent[] }
  | { type: typeof StoreAction.ResetAuth }
  | { type: typeof StoreAction.ResetAuthCheck }
  | { type: typeof StoreAction.SetAuth; value: Auth }
  | { type: typeof StoreAction.SetAuthCheck }
  | { type: typeof StoreAction.SetInfo; value: DeterminedInfo }
  | { type: typeof StoreAction.SetInfoCheck }
  | { type: typeof StoreAction.SetUsers; value: DetailedUser[] }
  | { type: typeof StoreAction.SetCurrentUser; value: DetailedUser }
  | { type: typeof StoreAction.SetUserSettings; value: V1UserWebSetting[] }
  | { type: typeof StoreAction.SetResourcePools; value: ResourcePool[] }
  | { type: typeof StoreAction.SetPinnedWorkspaces; value: Workspace[] }
  | { type: typeof StoreAction.HideOmnibar }
  | { type: typeof StoreAction.ShowOmnibar }
  | {
      type: typeof StoreAction.SetActiveTasks;
      value: {
        commands: number;
        notebooks: number;
        shells: number;
        tensorboards: number;
      };
    }
  | { type: typeof StoreAction.SetActiveExperiments; value: number }
  | { type: typeof StoreAction.SetKnownRoles; value: UserRole[] }
  | { type: typeof StoreAction.SetUserRoles; value: UserRole[] }
  | { type: typeof StoreAction.SetUserAssignments; value: UserAssignment[] };

export const AUTH_COOKIE_KEY = 'auth';

const initAuth = {
  checked: false,
  isAuthenticated: false,
};
const initResourceTally: ClusterOverviewResource = { allocation: 0, available: 0, total: 0 };
const initClusterOverview: ClusterOverview = {
  [ResourceType.CPU]: clone(initResourceTally),
  [ResourceType.CUDA]: clone(initResourceTally),
  [ResourceType.ROCM]: clone(initResourceTally),
  [ResourceType.ALL]: clone(initResourceTally),
  [ResourceType.UNSPECIFIED]: clone(initResourceTally),
};
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
  activeExperiments: 0,
  activeTasks: {
    commands: 0,
    notebooks: 0,
    shells: 0,
    tensorboards: 0,
  },
  agents: [],
  auth: initAuth,
  cluster: initClusterOverview,
  info: initInfo,
  knownRoles: [],
  pinnedWorkspaces: [],
  pool: {},
  resourcePools: [],
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
  userSettings: [],
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

export const agentsToOverview = (agents: Agent[]): ClusterOverview => {
  // Deep clone for render detection.
  const overview: ClusterOverview = clone(initClusterOverview);

  agents.forEach((agent) => {
    agent.resources
      .filter((resource) => resource.enabled)
      .forEach((resource) => {
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
    overview[rt].allocation =
      overview[rt].total !== 0
        ? percent((overview[rt].total - overview[rt].available) / overview[rt].total)
        : 0;
  }

  return overview;
};

export const agentsToPoolOverview = (agents: Agent[]): PoolOverview => {
  const overview: PoolOverview = {};
  agents.forEach((agent) => {
    agent.resourcePools.forEach((pname) => {
      overview[pname] = clone(initResourceTally);
      agent.resources
        .filter((resource) => resource.enabled)
        .forEach((resource) => {
          const isResourceFree = resource.container == null;
          const availableResource = isResourceFree ? 1 : 0;
          overview[pname].available += availableResource;
          overview[pname].total += 1;
        });
    });
  });

  for (const key in overview) {
    overview[key].allocation =
      overview[key].total !== 0
        ? percent((overview[key].total - overview[key].available) / overview[key].total)
        : 0;
  }

  return overview;
};

// TODO turn this into a partial reducer simliar to reducerUI.
const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case StoreAction.Reset:
      return clone(initState) as State;
    case StoreAction.SetAgents: {
      if (isEqual(state.agents, action.value)) return state;
      const cluster = agentsToOverview(action.value);
      const pool = agentsToPoolOverview(action.value);
      return { ...state, agents: action.value, cluster, pool };
    }
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
    case StoreAction.SetInfo:
      if (isEqual(state.info, action.value)) return state;
      return { ...state, info: action.value };
    case StoreAction.SetInfoCheck:
      return { ...state, info: { ...state.info, checked: true } };
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
    case StoreAction.SetUserSettings:
      if (isEqual(state.userSettings, action.value)) return state;
      return { ...state, userSettings: action.value };
    case StoreAction.SetResourcePools:
      if (isEqual(state.resourcePools, action.value)) return state;
      return { ...state, resourcePools: action.value };
    case StoreAction.SetPinnedWorkspaces:
      if (isEqual(state.pinnedWorkspaces, action.value)) return state;
      return { ...state, pinnedWorkspaces: action.value };
    case StoreAction.HideOmnibar:
      if (!state.ui.omnibar.isShowing) return state;
      return { ...state, ui: { ...state.ui, omnibar: { ...state.ui.omnibar, isShowing: false } } };
    case StoreAction.ShowOmnibar:
      if (state.ui.omnibar.isShowing) return state;
      return { ...state, ui: { ...state.ui, omnibar: { ...state.ui.omnibar, isShowing: true } } };
    case StoreAction.SetActiveExperiments:
      if (isEqual(state.activeExperiments, action.value)) return state;
      return { ...state, activeExperiments: action.value };
    case StoreAction.SetActiveTasks:
      if (isEqual(state.activeTasks, action.value)) return state;
      return { ...state, activeTasks: action.value };
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
