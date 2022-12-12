import React, { Dispatch, useContext, useReducer } from 'react';

import { StoreProvider as UIStoreProvider } from 'shared/contexts/stores/UI';
import { clone, isEqual } from 'shared/utils/data';
import rootLogger from 'shared/utils/Logger';
import { checkDeepEquality } from 'shared/utils/store';
import { UserAssignment, UserRole } from 'types';

const logger = rootLogger.extend('store');

interface Props {
  children?: React.ReactNode;
}
interface State {
  knownRoles: UserRole[];
  userAssignments: UserAssignment[];
}

export const StoreAction = {
  Reset: 'Reset',

  // User assignments, roles, and derived permissions
  SetKnownRoles: 'SetKnownRoles',

  // User Settings
  SetUserSettings: 'SetUserSettings',
} as const;

type Action =
  | { type: typeof StoreAction.Reset }
  | { type: typeof StoreAction.SetKnownRoles; value: UserRole[] };

const initState: State = {
  knownRoles: [],
  userAssignments: [],
};

const StateContext = React.createContext<State | undefined>(undefined);
const DispatchContext = React.createContext<Dispatch<Action> | undefined>(undefined);

// TODO turn this into a partial reducer simliar to reducerUI.
const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case StoreAction.Reset:
      return clone(initState) as State;
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
