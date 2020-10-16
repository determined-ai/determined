import axios, { AxiosResponse, CancelToken } from 'axios';

import handleError, { DaError, ErrorLevel, ErrorType, isDaError } from 'ErrorHandler';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import { isObject } from 'utils/data';

import { DetApi, HttpApi } from './types';

/* eslint-disable @typescript-eslint/no-var-requires */
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

/* HTTP Helpers */

export const http = axios.create({ responseType: 'json', withCredentials: true });

export const processApiError = (name: string, e: Error): DaError => {
  const isAuthError = isAuthFailure(e);
  const silent = !process.env.IS_DEV || isAuthError || axios.isCancel(e);
  if (isDaError(e)) {
    if (e.type === ErrorType.ApiBadResponse) {
      e.message = `failed in decoding ${name} API response`;
      e.publicMessage = 'Failed to interpret data sent from the server.';
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
  return async function(params: Input & { cancelToken?: CancelToken }): Promise<Output> {
    const httpOpts = api.httpOptions(params);

    try {
      const response = api.stubbedResponse ? { data: api.stubbedResponse } as AxiosResponse<unknown>
        : await http.request({
          cancelToken: params.cancelToken,
          data: httpOpts.body,
          headers: httpOpts.headers,
          method: httpOpts.method || 'GET',
          url: httpOpts.url as string,
        });

      return api.postProcess ? api.postProcess(response) : response.data as Output;
    } catch (e) {
      processApiError(api.name, e);
      throw e;
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
    Api.ExperimentsApiFetchParamCreator().determinedTrialLogs(1, undefined, undefined, true),
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

    const response = await fetch(serverAddress(true, fetchArgs.url), options);
    const reader = ndjsonStream(response.body).getReader();
    let result;
    while (!result || !result.done) {
      result = await reader.read();
      if (result.done) return;
      onEvent(result.value.result);
    }
  } catch (e) {
    processApiError(fetchArgs.url, e);
    throw e;
  }
};

/* eslint-disable-next-line */
export const noOp = (): void => {}

export function generateDetApi<Input, DetOutput, Output>(api: DetApi<Input, DetOutput, Output>) {
  return async function(params: Input & { cancelToken?: CancelToken }): Promise<Output> {
    try {
      const response = api.stubbedResponse ? api.stubbedResponse : await api.request(params);
      return api.postProcess(response);
    } catch (e) {
      processApiError(api.name, e);
      throw e;
    }
  };
}
