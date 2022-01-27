import axios, { AxiosResponse } from 'axios';

import { globalStorage } from 'globalStorage';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import { isObject } from 'utils/data';
import { DetError, DetErrorOptions, ErrorLevel, ErrorType, isDetError } from 'utils/error';

import { ApiCommonParams, DetApi, FetchOptions, HttpApi } from './types';

/* eslint-disable @typescript-eslint/no-var-requires */
const ndjsonStream = require('can-ndjson-stream');

/* Response Helpers */

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
const getResponseStatus = (e: any): number | undefined => {
  const errorResponse = e || {};
  return (errorResponse.response || {}).status || errorResponse.status;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isAuthFailure = (e: any, supportExternalAuth = false): boolean => {
  const status = getResponseStatus(e) ?? 0;
  const authFailureStatuses = [ 401 ];
  if (supportExternalAuth) authFailureStatuses.push(500);
  return authFailureStatuses.includes(status);
};

const isApiResponse = (o: unknown): o is Response => {
  return o instanceof Response;
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
  return e?.name === 'AbortError' || axios.isCancel(e);
};

/* HTTP Helpers */

export const http = axios.create({ responseType: 'json', withCredentials: false });

/* Fits API errors into a DetError. */
export const processApiError = async (name: string, e: unknown): Promise<DetError> => {
  const isAuthError = isAuthFailure(e);
  const isApiBadResponse = isDetError(e) && e?.type === ErrorType.ApiBadResponse;
  const options: DetErrorOptions = {
    level: ErrorLevel.Error,
    publicSubject: `Request ${name} failed.`,
    silent: !process.env.IS_DEV || isAuthError || isAborted(e),
    type: ErrorType.Server,
  };

  if (isAuthError) {
    options.publicSubject = `Unauthenticated ${name} request.`;
    options.level = ErrorLevel.Fatal;
    options.type = ErrorType.Auth;
  } else if (isApiBadResponse) {
    options.publicSubject = `Failed in decoding ${name} API response.`;
  }

  if (isApiResponse(e)) {
    try {
      const response = await e.json();
      options.publicMessage = response.error?.error || response.message;
    } catch (err) {
      options.payload = err;
    }
  }
  return new DetError(e, options);
};

export function generateApi<Input, Output>(api: HttpApi<Input, Output>) {
  return async function(params: Input & ApiCommonParams): Promise<Output> {
    const httpOpts = api.httpOptions(params);

    try {
      let headers = httpOpts.headers;
      if (!api.unAuthenticated) {
        headers = {
          Authorization: `Bearer ${globalStorage.authToken}`,
          ...headers,
        };
      }
      const response = api.stubbedResponse ? { data: api.stubbedResponse } as AxiosResponse<unknown>
        : await http.request({
          cancelToken: params.cancelToken,
          data: httpOpts.body,
          headers,
          method: httpOpts.method || 'GET',
          url: serverAddress() + httpOpts.url,
        });

      return api.postProcess ? api.postProcess(response) : response.data as Output;
    } catch (e) {
      throw (await processApiError(api.name, e));
    }
  };
}

export function generateDetApi<Input, DetOutput, Output>(api: DetApi<Input, DetOutput, Output>) {
  return async function(params: Input, options?: FetchOptions): Promise<Output> {
    try {
      const response = api.stubbedResponse ?
        api.stubbedResponse : await api.request(params, options);
      return api.postProcess(response);
    } catch (e) {
      throw (await processApiError(api.name, e));
    }
  };
}

/* gRPC Helpers */

/*
  consumeStream is used to consume streams from the generated TS client.
  We use the provided fetchParamCreator to create fetch arguments and use that
  to make a request and handle events one by one.
  Example:
  consumeStream<Api.V1TrialLogsResponse>(
    Api.ExperimentsApiFetchParamCreator().trialLogs(1, undefined, undefined, true),
    console.log,
  ).then(() => console.log('finished'));
*/
export const consumeStream = async <T = unknown>(
  fetchArgs: Api.FetchArgs,
  onEvent: (event: T) => void,
): Promise<void> => {
  try {
    const options = isObject(fetchArgs.options) ? fetchArgs.options : {};

    /*
     * Default fetch credentials is set to `same-origin`, but we need to change it
     * to `include` for local dev because the ports do not match up (3000 vs 8080).
     */
    if (process.env.IS_DEV) options.credentials = 'include';

    const response = await fetch(serverAddress(fetchArgs.url), options);
    const reader = ndjsonStream(response.body).getReader();

    // Cancel reader if an abort signal is received.
    if (options?.signal) {
      const signal: AbortSignal = options.signal;
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
    const err = await processApiError(fetchArgs.url, e);
    if (!isAborted(e)) throw err;
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
export const identity = <T>(a: T): T => a;
