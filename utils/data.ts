/* eslint-disable @typescript-eslint/no-explicit-any */
import { Primitive, RawJson, RecordKey, UnknownRecord } from '../types';

// `bigint` is not support yet for
export const isBigInt = (data: unknown): data is bigint => typeof data === 'bigint';
export const isBoolean = (data: unknown): data is boolean => typeof data === 'boolean';
export const isDate = (data: unknown): data is Date => data instanceof Date;
export const isMap = (data: unknown): boolean => data instanceof Map;
export const isNullOrUndefined = (data: unknown): data is null | undefined => data == null;
export const isNumber = (data: unknown): data is number => typeof data === 'number';
export const isObject = (data: unknown): boolean => {
  return typeof data === 'object' && !Array.isArray(data) && !isSet(data) && data !== null;
};
export const isPrimitive = (data: unknown): boolean => (
  isBigInt(data) ||
  isBoolean(data) ||
  isNullOrUndefined(data) ||
  isNumber(data) ||
  isString(data) ||
  isSymbol(data)
);
export const isPromise = (data: unknown): data is Promise<unknown> => {
  if (!isObject(data)) return false;
  return typeof (data as { then?: any }).then === 'function';
};
export const isSet = (data: unknown): boolean => data instanceof Set;
export const isString = (data: unknown): data is string => typeof data === 'string';
export const isSymbol = (data: unknown): data is symbol => typeof data === 'symbol';
export const isFunction = (fn: unknown): boolean => typeof fn === 'function';

export const isAsyncFunction = (fn: unknown): boolean => {
  if (!isFunction(fn)) return false;
  return (fn as Promise<unknown>)[Symbol.toStringTag] === 'AsyncFunction';
};

export const isSyncFunction = (fn: unknown): boolean => {
  return isFunction(fn) && !isAsyncFunction(fn);
};

export const isEqual = (a: unknown, b: unknown): boolean => {
  if ((isMap(a) || isSet(b)) && (isMap(b) || isSet(b))) {
    return JSON.stringify(Array.from(a as any)) === JSON.stringify(Array.from(b as any));
  }
  if (isSymbol(a) && isSymbol(b)) return a.toString() === b.toString();
  if (isObject(a) && isObject(b)) return JSON.stringify(a) === JSON.stringify(b);
  if (Array.isArray(a) && Array.isArray(b))
    return a.length === b.length && a.every((x, i) => isEqual(x, b[i]));
  return a === b;
};

export const clone = (data: any, deep = true): any => {
  if (isPrimitive(data)) return data;
  if (isMap(data)) return new Map(data);
  if (isSet(data)) return new Set(data);
  return deep ? JSON.parse(JSON.stringify(data)) : { ...data };
};

export const hasObjectKeys = (data: unknown): boolean => {
  return isObject(data) && Object.keys(data as Record<RecordKey, unknown>).length !== 0;
};

export const flattenObject = <T = Primitive>(
  object: UnknownRecord,
  options?: {
    continueFn?: (value: unknown) => boolean,
    delimiter?: string,
    keys?: RecordKey[],
  },
): Record<RecordKey, T> => {
  const continueFn = options?.continueFn ?? isObject;
  const delimiter = options?.delimiter ?? '.';
  const keys = options?.keys ?? [];
  return Object.keys(object).reduce((acc, key) => {
    const value = object[key] as UnknownRecord;
    const newKeys = [ ...keys, key ];
    if (continueFn(value)) {
      acc = { ...acc, ...flattenObject<T>(value, { continueFn, delimiter, keys: newKeys }) };
    } else {
      const keyPath = newKeys.join(delimiter);
      acc[keyPath] = value as T;
    }
    return acc;
  }, {} as Record<RecordKey, T>);
};

export const unflattenObject = <T = unknown>(
  object: Record<RecordKey, T>,
  delimiter = '.',
): UnknownRecord => {
  const unflattened: UnknownRecord = {};
  const regexSafeDelimiter = delimiter.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  const regex = new RegExp(`^(.+?)${regexSafeDelimiter}(.+)$`);
  Object.entries(object).forEach(([ paramPath, value ]) => {
    let key = paramPath;
    let matches = key.match(regex);
    let pathRef = unflattened;
    while (matches?.length === 3) {
      const prefix = matches[1];
      key = matches[2];
      pathRef[prefix] = pathRef[prefix] ?? {};
      pathRef = pathRef[prefix] as UnknownRecord;
      matches = key.match(regex);
    }
    pathRef[key] = value;
  });
  return unflattened;
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

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const validateEnum = (enumObject: unknown, value?: unknown): any => {
  if (isObject(enumObject) && value !== undefined) {
    const enumRecord = enumObject as Record<string, string>;
    const stringValue = value as string;
    const validOptions = Object.values(enumRecord);
    if (validOptions.includes(stringValue)) return stringValue;
  }
  return undefined;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const validateEnumList = (enumObject: unknown, values?: unknown[]): any => {
  if (!Array.isArray(values)) return undefined;

  const enumValues = values
    .map(value => validateEnum(enumObject, value))
    .filter(value => !!value);

  return enumValues.length !== 0 ? enumValues : undefined;
};
