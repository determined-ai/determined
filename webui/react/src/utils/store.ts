import _ from 'lodash';
import { Reducer } from 'react';

import { LoggerInterface } from 'utils/Logger';

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
    if (_.isEqual(state, newState)) return state;
    logger?.debug('store state updated', action.type);
    return newState;
  };
};

// eslint-disable-next-line  @typescript-eslint/no-explicit-any
export const encodeParams = (params: { [key: string]: any }): string =>
  JSON.stringify([...Object.entries(params ?? {})].sort());
