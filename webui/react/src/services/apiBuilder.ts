import axios, { AxiosResponse, CancelToken, Method } from 'axios';

import handleError, { DaError, ErrorLevel, ErrorType, isDaError } from 'ErrorHandler';
import { isAuthFailure } from 'services/api';
import * as DetSwagger from 'services/api-ts-sdk';

/* eslint-disable @typescript-eslint/no-var-requires */
const ndjsonStream = require('can-ndjson-stream');

export interface HttpOptions {
  url?: string;
  method?: Method;
  headers?: Record<string, unknown>;
  body?: Record<keyof unknown, unknown> | string;
}

export interface Api<Input, Output>{
  name: string;
  httpOptions: (params: Input) => HttpOptions;
  postProcess?: (response: AxiosResponse<unknown>) => Output; // io type decoder.
  stubbedResponse?: unknown; // optional stubbed response body.
  // middlewares?: Middleware[]; // success/failure middlewares
}

export const http = axios.create({
  responseType: 'json',
  withCredentials: true,
});

export const serverAddress = (avoidDevProxy = false): string => {
  if (avoidDevProxy && process.env.IS_DEV) return 'http://localhost:8080';
  return `${window.location.protocol}//${window.location.host}`;
};

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

export function generateApi<Input, Output>(api: Api<Input, Output>) {
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

/*
  consumeStream is used to consume streams from the generated TS client.
  We use the provided fetchParamCreator to create fetch arguments and use that
  to make a request and handle events one by one.
  Example:
  consumeStream<DetSwagger.V1TrialLogsResponse>(
    DetSwagger.ExperimentsApiFetchParamCreator().determinedTrialLogs(1, undefined, undefined, true),
    console.log,
  ).then(() => console.log('finished'));
*/
export const consumeStream = async <T = unknown>(
  fetchArgs: DetSwagger.FetchArgs, onEvent: (event: T) => void): Promise<void> => {
  try {
    const response = await fetch(serverAddress(true) + fetchArgs.url, fetchArgs.options);
    const exampleReader = ndjsonStream(response.body).getReader();
    let result;
    while (!result || !result.done) {
      result = await exampleReader.read();
      if (result.done) return;
      onEvent(result.value.result);
    }
  } catch (e) {
    processApiError(fetchArgs.url, e);
    throw e;
  }
};
