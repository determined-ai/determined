import axios from 'axios';
import { Dispatch, Reducer, SetStateAction, useEffect, useReducer, useState } from 'react';

import { ApiState } from 'services/types';
import { clone, isEqual } from 'utils/data';

export enum ActionType {
  SetData,
  SetError,
  SetLoading,
}

export interface RestApiState<T> extends ApiState<T> {
  errorCount: number;
  hasLoaded: boolean;
}

type Action<T> =
  | { type: ActionType.SetData; value: T }
  | { type: ActionType.SetError; value: Error }
  | { type: ActionType.SetLoading; value: boolean }

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
type Mapper = (x: any) => any;

type Output<In, Out> = [
  RestApiState<Out>,
  Dispatch<SetStateAction<In>>,
];

const defaultReducer = <T>(state: RestApiState<T>, action: Action<T>): RestApiState<T> => {
  switch (action.type) {
    case ActionType.SetData: {
      const data = isEqual(action.value, state.data) ? state.data : action.value;
      return { ...state, data, hasLoaded: true, isLoading: false };
    }
    case ActionType.SetError:
      return {
        ...state,
        error: action.value,
        errorCount: state.errorCount + 1,
        isLoading: false,
      };
    case ActionType.SetLoading:
      return { ...state, isLoading: action.value };
    default:
      return state;
  }
};

export const applyMappers = <T>(data: unknown, mappers: Mapper | Mapper[]): T => {
  let currentData = clone(data);

  if (Array.isArray(mappers)) {
    currentData = mappers.reduce((acc, mapper) => mapper(acc), currentData);
  } else {
    currentData = mappers(currentData);
  }

  return currentData;
};

const useRestApi = <In, Out>(
  apiRequest: (a: In) => Promise<Out>,
  initialParams: In,
): Output<In, Out> => {
  const [ params, setParams ] = useState<In>(initialParams);
  const [ state, dispatch ] = useReducer<Reducer<RestApiState<Out>, Action<Out>>>(defaultReducer, {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  });

  useEffect(() => {
    const source = axios.CancelToken.source();

    dispatch({ type: ActionType.SetLoading, value: true });
    apiRequest({ ...params, cancelToken: source.token })
      .then((result) => dispatch({ type: ActionType.SetData, value: result }))
      .catch((e) => (!axios.isCancel(e)) && dispatch({ type: ActionType.SetError, value: e }));

    return (): void => source.cancel();
  }, [ apiRequest, params ]);

  return [ state, setParams ];
};

export default useRestApi;
