import axios, { AxiosResponse, CancelToken, Method } from 'axios';

import { crossoverRoute } from 'routes';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
const hasAuthFailed = (e: any): boolean => {
  return e.response && e.response.status && e.response.status === 401;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
const handleAuthFailure = (e: any): boolean => {
  if (!hasAuthFailed(e)) return false;

  // TODO: Update to internal routing when React takes over login.
  crossoverRoute('/ui/logout');
  return true;
};

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
  /* eslint-disable-next-line @typescript-eslint/no-explicit-any */
  postProcess?: (response: AxiosResponse<any>) => Output; // io type decoder.
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

      return api.postProcess ? api.postProcess(response) : response.data as unknown as Output;
    } catch (e) {
      handleAuthFailure(e);
      throw Error(`${api.name} failed`);
    }
  };
}
