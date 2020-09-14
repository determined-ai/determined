import axios, { AxiosResponse, CancelToken } from 'axios';

import handleError, { DaError, ErrorLevel, ErrorType, isDaError } from 'ErrorHandler';
import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';

import { HttpApi } from './types';

/* eslint-disable @typescript-eslint/no-var-requires */
const ndjsonStream = require('can-ndjson-stream');

/* Response Helpers */

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isAuthFailure = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 401;
};

// is a failure received from a failed login attempt due to bad credentials
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isLoginFailure = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 403;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isNotFound = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 404;
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
    const response = await fetch(serverAddress(true, fetchArgs.url), fetchArgs.options);
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
