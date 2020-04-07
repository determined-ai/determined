export const isPrimitive = <T>(data: T): boolean => data !== Object(data);
export const isMap = <T>(data: T): boolean => data instanceof Map;
export const isNumber = <T>(data: T): boolean => typeof data === 'number';
export const isSet = <T>(data: T): boolean => data instanceof Set;

export const isFunction = (fn: unknown): boolean => {
  return typeof fn === 'function';
};

export const isAsyncFunction = (fn: unknown): boolean => {
  if (!isFunction(fn)) return false;
  return (fn as Promise<unknown>)[Symbol.toStringTag] === 'AsyncFunction';
};

export const isSyncFunction = (fn: unknown): boolean => {
  return !isAsyncFunction(fn);
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const clone = (data: any, deep = true): any => {
  if (isPrimitive(data)) return data;
  if (isMap(data)) return new Map(data);
  if (isSet(data)) return new Set(data);
  return deep ? JSON.parse(JSON.stringify(data)) : { ...data };
};
