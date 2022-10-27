import { Reducer } from 'react';

import { isEqual } from '../utils/data';

import { LoggerInterface } from './Logger';

/** has a printable type */
export interface TypeWise {
  type: string | number;
}

/**
 * wrap a reducer to allow catching unecessary updates
 * based on a deep equality check.
 */
export const checkDeepEquality = <State, Action extends TypeWise>(
  reducer: Reducer<State, Action>,
  logger?: LoggerInterface,
) => {
  return (state: State, action: Action): State => {
    const newState = reducer(state, action);
    if (isEqual(state, newState)) return state;
    logger?.debug('store state updated', action.type);
    return newState;
  };
};
