import { RawJson } from 'types';

export const isMap = <T>(data: T): boolean => data instanceof Map;
export const isBoolean = (data: unknown): boolean => typeof data === 'boolean';
export const isNumber = (data: unknown): data is number => typeof data === 'number';
export const isObject = <T>(data: T): boolean => typeof data === 'object' && data !== null;
export const isPrimitive = <T>(data: T): boolean => data !== Object(data);
export const isSet = <T>(data: T): boolean => data instanceof Set;
export const isFunction = (fn: unknown): boolean => {
  return typeof fn === 'function';
};

export const isAsyncFunction = (fn: unknown): boolean => {
  if (!isFunction(fn)) return false;
  return (fn as Promise<unknown>)[Symbol.toStringTag] === 'AsyncFunction';
};

export const isSyncFunction = (fn: unknown): boolean => {
  return isFunction(fn) && !isAsyncFunction(fn);
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const clone = (data: any, deep = true): any => {
  if (isPrimitive(data)) return data;
  if (isMap(data)) return new Map(data);
  if (isSet(data)) return new Set(data);
  return deep ? JSON.parse(JSON.stringify(data)) : { ...data };
};

export const getPath = <T>(obj: RawJson, path: string): T | undefined => {
  // Reassigns to obj[key] on each array.every iteration
  if (path === '') return obj as T;
  let value = obj || {};
  return path.split('.').every(key => ((value = value[key]) !== undefined)) ?
    value as T : undefined;
};

export const getPathList = <T>(obj: RawJson, path: string[]): T | undefined => {
  return getPath<T>(obj, path.join('.'));
};

export const getPathOrElse = <T>(
  obj: Record<string, unknown>,
  path: string,
  fallback: T,
): T => {
  const value = getPath<T>(obj, path);
  return value !== undefined ? value : fallback;
};

export const categorize = <T>(array: T[], keyFn: ((arg0: T) => string)): Record<string, T[]> => {
  const d: Record<string, T[]> = {};
  array.forEach(item => {
    const key = keyFn(item);
    d[key] ? d[key].push(item) : d[key] = [ item ];
  });
  return d;
};

// We avoid exporting this type to discourage/disallow usage of any.
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
type Mapper = (x: any) => any;
export const applyMappers = <T>(data: unknown, mappers: Mapper | Mapper[]): T => {
  let currentData = clone(data);

  if (Array.isArray(mappers)) {
    currentData = mappers.reduce((acc, mapper) => mapper(acc), currentData);
  } else {
    currentData = mappers(currentData);
  }

  return currentData;
};

export const isEqual = (a: unknown, b: unknown): boolean => {
  if (a === b) return true;
  return JSON.stringify(a) === JSON.stringify(b);
};

export const setPathList = (obj: RawJson, path: string[], value: unknown): void => {
  const lastIndex = path.length - 1;
  const parentObj = getPathList<RawJson>(obj, path.slice(0, lastIndex));
  if (parentObj) parentObj[path[lastIndex]] = value;
};

export const deletePathList = (obj: RawJson, path: string[]): void => {
  const lastIndex = path.length - 1;
  const parentObj = getPathList<RawJson>(obj, path.slice(0, lastIndex));
  if (parentObj) delete parentObj[path[lastIndex]];
};
