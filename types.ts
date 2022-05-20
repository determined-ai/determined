import { RouteProps } from 'react-router';

export type Primitive = boolean | number | string;
export type RecordKey = string | number | symbol;
export type UnknownRecord = Record<RecordKey, unknown>;
export type NullOrUndefined<T = undefined> = T | null | undefined;
export type Point = { x: number; y: number };
export type Range<T = Primitive> = [ T, T ];
export type Eventually<T> = T | Promise<T>;
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export type RawJson = Record<string, any>;

export interface Pagination {
  limit: number;
  offset: number;
}

export interface FetchOptions {
  signal?: AbortSignal;
}

interface ApiBase {
  name: string;
  stubbedResponse?: unknown;
  unAuthenticated?: boolean;
  // middlewares?: Middleware[]; // success/failure middlewares
}

// Designed for use with Swagger generated api bindings.
export interface DetApi<Input, DetOutput, Output> extends ApiBase {
  postProcess: (response: DetOutput) => Output;
  request: (params: Input, options?: FetchOptions) => Promise<DetOutput>;
  stubbedResponse?: DetOutput;
}

export interface ApiState<T> {
  data?: T;
  error?: Error;
  isLoading: boolean;
}

export interface SingleEntityParams {
  id: number;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
export type EmptyParams = {}

/*
 * Router Configuration
 * If the component is not defined, the route is assumed to be an external route,
 * meaning React will attempt to load the path outside of the internal routing
 * mechanism.
 */
export interface RouteConfig extends RouteProps {
  icon?: string;
  id: string;
  needAuth?: boolean;
  path: string;
  popout?: boolean;
  redirect?: string;
  suffixIcon?: string;
  title?: string;
}

export type CommonProps = {
  children?: React.ReactNode;
  className?: string;
  title?: string;
};
