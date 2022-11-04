import { RouteProps } from 'react-router-dom';

export type Primitive = boolean | number | string;
export type RecordKey = string | number | symbol;
export type UnknownRecord = Record<RecordKey, unknown>;
export type NullOrUndefined<T = undefined> = T | null | undefined;
export type Point = { x: number; y: number };
export type Range<T = Primitive> = [T, T];
export type Eventually<T> = T | Promise<T>;
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export type RawJson = Record<string, any>;

export interface Pagination {
  limit: number;
  offset: number;
  total?: number;
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

export type RecordUnknown = Record<RecordKey, unknown>;

// Designed for use with Swagger generated api bindings.
export interface DetApi<Input, DetOutput, Output> extends ApiBase {
  postProcess: (response: DetOutput) => Output;
  request: (params: Input, options?: FetchOptions) => Promise<DetOutput>;
  stubbedResponse?: DetOutput;
}

/**
 * @description helper to organize storing api response data.
 */
export interface ApiState<T> {
  data?: T;
  /**
   * error, if any, with the last state update.
   * this should be cleared on the next successful update.
   */
  error?: Error;
  /**
   * indicates whether the state has been fetched at least once or not.
   * should always be initialized to false.
   */
  hasBeenInitialized?: boolean;
  /** is the state being updated? */
  isLoading?: boolean;
}

export interface SingleEntityParams {
  id: number;
}

/* eslint-disable-next-line @typescript-eslint/ban-types */
export type EmptyParams = {};

/**
 * Router Configuration
 * If the component is not defined, the route is assumed to be an external route,
 * meaning React will attempt to load the path outside of the internal routing
 * mechanism.
 */
export type RouteConfig = {
  icon?: string;
  id: string;
  needAuth?: boolean;
  path: string;
  popout?: boolean;
  redirect?: string;
  suffixIcon?: string;
  title?: string;
} & RouteProps;

export interface ClassNameProp {
  /** classname to be applied to the base element */
  className?: string;
}
export interface CommonProps extends ClassNameProp {
  children?: React.ReactNode;
  title?: string;
}

export interface SemanticVersion {
  major: number;
  minor: number;
  patch: number;
}

export type ValueOf<T> = T[keyof T];
