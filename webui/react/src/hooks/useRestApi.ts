import axios from 'axios';
import * as io from 'io-ts';
import { Dispatch, Reducer, SetStateAction, useEffect, useReducer, useState } from 'react';

import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { decode } from 'ioTypes';
import { isAuthFailure } from 'services/api';
import { http, HttpOptions } from 'services/apiBuilder';
import { clone } from 'utils/data';

enum ActionType {
  SetData,
  SetError,
  SetLoading,
}

type State<T> = {
  data?: T;
  error?: Error;
  errorCount: number;
  hasLoaded: boolean;
  isLoading: boolean;
};

export type RestApiState<T> = State<T>;

type Action<T> =
  | { type: ActionType.SetData; value: T }
  | { type: ActionType.SetError; value: Error }
  | { type: ActionType.SetLoading; value: boolean }

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
type Mapper = (x: any) => any;

interface HookOptions<T> {
  httpOptions?: HttpOptions;
  data?: T;
  mappers?: Mapper | Mapper[];
}

type Output<T> = [
  State<T>,
  Dispatch<SetStateAction<HttpOptions>>,
];

const reducer = <T>(state: State<T>, action: Action<T>): State<T> => {
  switch (action.type) {
    case ActionType.SetData:
      return { ...state, data: action.value, hasLoaded: true, isLoading: false };
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

const useRestApi = <T>(ioType: io.Mixed, options: HookOptions<T> = {}): Output<T> => {
  const [ httpOptions, setHttpOptions ] = useState<HttpOptions>(options.httpOptions || {});
  const [ state, dispatch ] = useReducer<Reducer<State<T>, Action<T>>>(reducer, {
    data: options.data,
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  });

  useEffect(() => {
    const source = axios.CancelToken.source();
    if (!httpOptions.url) return;
    httpOptions.method = httpOptions.method || 'GET';

    const fetchData = async (): Promise<void> => {
      dispatch({ type: ActionType.SetLoading, value: true });

      try {
        const response = await http.request({
          cancelToken: source.token,
          data: httpOptions.body,
          method: httpOptions.method,
          url: httpOptions.url as string,
        });
        const result = decode<io.TypeOf<typeof ioType>>(ioType, response.data);

        dispatch({
          type: ActionType.SetData,
          value: options.mappers ? applyMappers(result, options.mappers) : result,
        });
      } catch (error) {
        // Only report errors not related cancel exits.
        if (!axios.isCancel(error)) {
          handleError({
            error: error,
            // this does not necessarily have to be true for all usages of this hook we should
            // allow the user of the hook to set this value or let the caller handle the error.
            isUserTriggered: false,
            level: ErrorLevel.Warn,
            message: `${httpOptions.method} request to ${httpOptions.url} failed`,
            type: isAuthFailure(error) ? ErrorType.Auth : ErrorType.Server,
          });

          dispatch({ type: ActionType.SetError, value: error });
        }
      }
    };

    fetchData();

    return (): void => source.cancel();
  }, [ httpOptions, ioType, options.mappers ]);

  return [ state, setHttpOptions ];
};

type SimpleOutput<In, Out> = [
  State<Out>,
  Dispatch<SetStateAction<In>>,
];

export const useRestApiSimple =
<In, Out>(apiReq: (a: In) => Promise<Out>, initialParams: In): SimpleOutput<In, Out> => {
  const [ params, setParams ] = useState<In>(initialParams);
  const [ state, dispatch ] = useReducer<Reducer<State<Out>, Action<Out>>>(reducer, {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  });

  useEffect(() => {
    const source = axios.CancelToken.source();

    apiReq({ ...params, cancelToken: source.token })
      .then((result) => dispatch({ type: ActionType.SetData, value: result } ))
      .catch((e) => (!axios.isCancel(e)) && dispatch({ type: ActionType.SetError, value: e }));

    return (): void => source.cancel();
  }, [ apiReq, params ]);

  return [ state, setParams ];
};

export default useRestApi;
