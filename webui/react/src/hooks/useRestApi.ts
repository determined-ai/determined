import axios, { Method } from 'axios';
import * as io from 'io-ts';
import { Dispatch, Reducer, SetStateAction, useEffect, useReducer, useState } from 'react';

import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { decode } from 'ioTypes';
import { http } from 'services/api';
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

interface HttpOptions {
  url?: string;
  method?: Method;
  body?: Record<keyof unknown, unknown> | string;
}

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
  const source = axios.CancelToken.source();
  const [ httpOptions, setHttpOptions ] = useState<HttpOptions>(options.httpOptions || {});
  const [ state, dispatch ] = useReducer<Reducer<State<T>, Action<T>>>(reducer, {
    data: options.data,
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  });

  useEffect(() => {
    if (!httpOptions.url) return;

    // cancel the previous existing request
    if (state.isLoading) source.cancel();

    const fetchData = async (): Promise<void> => {
      dispatch({ type: ActionType.SetLoading, value: true });

      try {
        const response = await http.request({
          cancelToken: source.token,
          data: httpOptions.body,
          method: httpOptions.method || 'GET',
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
            // this does not necessarily have to be true for all usages of this hook we should
            // allow the user of the hook to set this value or let the caller handle the error.
            isUserTriggered: false,
            level: ErrorLevel.Warn,
            message: `${httpOptions.method + ' ' || ''}request to ${httpOptions.url} failed`,
            type: ErrorType.Server,
          }, false);

          dispatch({ type: ActionType.SetError, value: error });
        }
      }
    };

    fetchData();

    return (): void => source.cancel();
  }, [ httpOptions ]);

  return [ state, setHttpOptions ];
};

export default useRestApi;
