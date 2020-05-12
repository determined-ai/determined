import axios, { AxiosResponse, CancelToken, Method } from 'axios';

import handleError, { ErrorLevel, ErrorType } from 'ErrorHandler';
import { isAuthFailure } from 'services/api';

export const http = axios.create({
  responseType: 'json',
  withCredentials: true,
});

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
  // middlewares?: Middleware[]; // success/failure middlewares
}

export function generateApi<Input, Output>(api: Api<Input, Output>) {
  return async function(params: Input & { cancelToken?: CancelToken }): Promise<Output> {
    const httpOpts = api.httpOptions(params);

    try {
      const response = await http.request({
        cancelToken: params.cancelToken,
        data: httpOpts.body,
        method: httpOpts.method || 'GET',
        url: httpOpts.url as string,
      });

      return api.postProcess ? api.postProcess(response) : response.data as Output;
    } catch (e) {
      const isAuthError = isAuthFailure(e);
      const error = handleError({
        error: e,
        level: isAuthError ? ErrorLevel.Fatal : ErrorLevel.Error,
        message: isAuthError ?
          `unauthenticated request ${api.name}` : `request ${api.name} failed.`,
        silent: true,
        type: isAuthError ? ErrorType.Auth : ErrorType.Server,
      });
      throw error;
    }
  };
}
