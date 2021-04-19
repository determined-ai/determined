import axios, { AxiosResponse } from 'axios';

import handleError, { DaError, ErrorLevel, ErrorType, isDaError } from 'ErrorHandler';
import { globalStorage } from 'globalStorage';
import { applyRRMiddleware, genRequestKey, rrState } from 'recordReplay';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import { isObject } from 'utils/data';

import { ApiCommonParams, DetApi, FetchOptions, HttpApi } from './types';

/* eslint-disable-next-line @typescript-eslint/no-var-requires */
const ndjsonStream = require('can-ndjson-stream');

/* Response Helpers */

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
const getResponseStatus = (e: any): number | undefined => {
  const errorResponse = e || {};
  return (errorResponse.response || {}).status || errorResponse.status;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isAuthFailure = (e: any): boolean => {
  return getResponseStatus(e) === 401;
};

/*
 * This is a failure received from a failed login attempt due to bad credentials
 * 403 is returned by the old API
 * 401 is returned by the new API. We can rely on isAuthFailure
 * when we completely migrate over to the new API.
 */
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isLoginFailure = (e: any): boolean => {
  const status = getResponseStatus(e);
  return status === 401 || status === 403;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isNotFound = (e: any): boolean => {
  return getResponseStatus(e) === 404;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isAborted = (e: any): boolean => {
  return e && e.name && e.name === 'AbortError'|| axios.isCancel(e);
};

/* HTTP Helpers */

export const http = axios.create({ responseType: 'json', withCredentials: false });

export const processApiError = (name: string, e: Error): DaError => {
  const isAuthError = isAuthFailure(e);
  const silent = !process.env.IS_DEV || isAuthError || axios.isCancel(e);
  if (isDaError(e)) {
    if (e.type === ErrorType.ApiBadResponse) {
      e.message = `failed in decoding ${name} API response`;
      e.publicMessage = 'Unexpected response from the server.';
      e.publicSubject = 'Unexpected API response';
      e.silent = silent;
    }
    return handleError(e);
  }
  return handleError({
    error: e,
    level: isAuthError ? ErrorLevel.Fatal : ErrorLevel.Error,
    message: isAuthError ?
      `unauthenticated request ${name}` : `request ${name} failed.`,
    silent,
    type: isAuthError ? ErrorType.Auth : ErrorType.Server,
  });
};
export function generateApi<Input, Output>(api: HttpApi<Input, Output>) {
  return async function(params: Input & ApiCommonParams): Promise<Output> {
    const httpOpts = api.httpOptions(params);

    const key = await genRequestKey(api.name, params); // TODO tack on axios
    try {
      const response = await applyRRMiddleware(key, async () => {
        let headers = httpOpts.headers;
        if (!api.unAuthenticated) {
          headers = {
            Authorization: `Bearer ${globalStorage.authToken}`,
            ...headers,
          };
        }
        return api.stubbedResponse ? { data: api.stubbedResponse }
          : await http.request({
            cancelToken: params.cancelToken,
            data: httpOpts.body,
            headers,
            method: httpOpts.method || 'GET',
            url: serverAddress() + httpOpts.url,
          });
      }) as AxiosResponse<unknown>;

      return api.postProcess ? api.postProcess(response) : response.data as Output;
    } catch (e) {
      processApiError(api.name, e);
      throw e;
    }
  };
}

export function generateDetApi<Input, DetOutput, Output>(api: DetApi<Input, DetOutput, Output>) {
  return async function(params: Input, options?: FetchOptions): Promise<Output> {
    try {
      const key = await genRequestKey(api.name, params);
      // FIXME cancel/abort fnctionality. also this might be better if it sits outsdie try/catch
      const response = await applyRRMiddleware<DetOutput>(key, () => {
        return api.stubbedResponse ?
          api.stubbedResponse : api.request(params, options);
      });
      return api.postProcess(response);
    } catch (e) {
      if (!isAborted(e)) processApiError(api.name, e);
      throw e;
    }
  };
}

/* gRPC Helpers */

const makeFetch = async (
  fetchArgs: Api.FetchArgs,
): Promise<Response> => {
  const options = isObject(fetchArgs.options) ? fetchArgs.options : {};

  /*
     * Default fetch credentials is set to `same-origin`, but we need to change it
     * to `include` for local dev because the ports do not match up (3000 vs 8080).
     */
  if (process.env.IS_DEV) options.credentials = 'include';

  const response = await fetch(serverAddress(fetchArgs.url), options);
  if (!(response instanceof Response)) {
    throw new Error('fetch failed'); // FIXME
  }
  return response;
};

interface EventValue<T> {
  error?: Error;
  result?: T
}

export const fetchStreamOneShot = async <T = unknown>(
  fetchArgs: Api.FetchArgs,
): Promise<EventValue<T>[]> => {
  const response = await makeFetch(fetchArgs);
  const resp = await response.text();
  // if (resp === '') return [];
  const values = resp.split('\n')
    .filter(l => l !== '')
    .map(line => {
      try {
        return JSON.parse(line);
      } catch (_) {
        return null;
      }
    }).filter(x => x);
  return values;
};

/*
  consumeStream is used to consume streams from the generated TS client.
  We use the provided fetchParamCreator to create fetch arguments and use that
  to make a request and handle events one by one.
  Example:
  consumeStream<Api.V1TrialLogsResponse>(
    Api.ExperimentsApiFetchParamCreator().determinedTrialLogs(1, undefined, undefined, true),
    console.log,
  ).then(() => console.log('finished'));
*/
export const consumeStream = async <T = unknown>(
  fetchArgs: Api.FetchArgs,
  onEvent: (event: T) => void,
): Promise<void> => {
  try {

    if (rrState.mode !== 'disabled') {
      // TODO use a custom recorder and record the same stream.
      const key = await genRequestKey(fetchArgs.url, {});
      // FIXME this still opens a connection to ther server..
      const values = await applyRRMiddleware(
        key,
        () => fetchStreamOneShot<T>(fetchArgs),
      ) as EventValue<T>[];
      values.filter(val => !!val.result && !val.error)
        .forEach(val => onEvent(val.result as T));
      return;
    }

    const response = await makeFetch(fetchArgs);
    const reader = ndjsonStream(response.body).getReader();

    // Cancel reader if an abort signal is received.
    if (fetchArgs.options && fetchArgs.options.signal) {
      const signal: AbortSignal = fetchArgs.options.signal;
      const abortHandler = () => {
        reader.cancel();
        signal.removeEventListener('abort', abortHandler);
      };
      signal.addEventListener('abort', abortHandler);
    }

    let result;
    while (!result || !result.done) {
      result = await reader.read();
      if (result.done) return;
      if (result.value.error) {
        throw result.value.error;
      } else {
        onEvent(result.value.result);
      }
    }
  } catch (e) {
    if (!isAborted(e)) {
      processApiError(fetchArgs.url, e);
      throw e;
    }
  }
};

/*
 * This function is primarily used to convert an enum option into a string value
 * that the generated API can take as a request param.
 * More specifically the function takes a value and checks it against a Typescript enum,
 * to make sure the the value is one of the enum option and returns the value as a string.
 */
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const validateDetApiEnum = (enumObject: unknown, value?: unknown): any => {
  if (isObject(enumObject) && value !== undefined) {
    const enumRecord = enumObject as Record<string, string>;
    const stringValue = value as string;
    const validOptions = Object
      .values(enumRecord)
      .filter((_, index) => index % 2 === 0);
    if (validOptions.includes(stringValue)) return stringValue;
    return enumRecord.UNSPECIFIED;
  }
  return undefined;
};

/*
 * This is the same as validateDetApiEnum but validates a list of values.
 * If the validated list is empty, this will return undefined because our
 * API will skip filtering if it sees an `undefined` value for a filter
 * query parameter.
 */
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const validateDetApiEnumList = (enumObject: unknown, values?: unknown[]): any => {
  if (!Array.isArray(values)) return undefined;

  const enumValues = values
    .map(value => validateDetApiEnum(enumObject, value))
    .filter(enumValue => enumValue !== (enumObject as { UNSPECIFIED: unknown }).UNSPECIFIED);
  return enumValues.length !== 0 ? enumValues : undefined;
};

/* eslint-disable-next-line */
export const noOp = (): void => {}
