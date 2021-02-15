import { CommandState, MetricName, MetricType, Primitive, RawJson, RunState, State } from 'types';

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

const runStateSortValues: Record<RunState, number> = {
  [RunState.Active]: 0,
  [RunState.Paused]: 1,
  [RunState.StoppingError]: 2,
  [RunState.Errored]: 3,
  [RunState.StoppingCompleted]: 4,
  [RunState.Completed]: 5,
  [RunState.StoppingCanceled]: 6,
  [RunState.Canceled]: 7,
  [RunState.Deleted]: 7,
  [RunState.Unspecified]: 8,
};

const commandStateSortValues: Record<CommandState, number> = {
  [CommandState.Pending]: 0,
  [CommandState.Assigned]: 1,
  [CommandState.Pulling]: 2,
  [CommandState.Starting]: 3,
  [CommandState.Running]: 4,
  [CommandState.Terminating]: 5,
  [CommandState.Terminated]: 6,
};

/* Sorters */

export const alphanumericSorter = (a: string | number, b: string | number): number => {
  return a.toString().localeCompare(b.toString(), 'en', { numeric: true });
};

export const booleanSorter = (a: boolean, b: boolean): number => {
  // True values first.
  return (a === b) ? 0 : (a ? -1 : 1);
};

export const commandStateSorter = (a: CommandState, b: CommandState): number => {
  return commandStateSortValues[a] - commandStateSortValues[b];
};

export const primitiveSorter = (a: Primitive, b: Primitive): number => {
  if (typeof a === 'boolean' && typeof b === 'boolean') {
    return booleanSorter(a, b);
  } else if (typeof a === 'number' && typeof b === 'number') {
    return numericSorter(a, b);
  } else if (typeof a === 'string' && typeof b === 'string') {
    return alphanumericSorter(a, b);
  }
  return 0;
};

/*
 * Sort the metric names by having the validation metrics come first followed by training metrics.
 * Within each type of metric, sort in the order they appear in the `MetricNames` array.
 * Within the respective type of metrics, `MetricNames` is currently sorted alphanumerically.
 */
export const metricNameSorter = (a: MetricName, b: MetricName): number => {
  const isAValidation = a.type === MetricType.Validation;
  const isBValidation = b.type === MetricType.Validation;
  if (isAValidation && !isBValidation) return -1;
  if (isBValidation && !isAValidation) return 1;
  return alphanumericSorter(a.name, b.name);
};

export const numericSorter = (a?: number, b?: number, reverseOrder = false): number => {
  if (a != null && b != null) {
    const diff = reverseOrder ? b - a : a - b;
    if (diff < 0) return -1;
    if (diff > 0) return 1;
    return 0;
  }
  if (a != null && b == null) return reverseOrder ? -1 : 1;
  if (a == null && b != null) return reverseOrder ? 1 : -1;
  return 0;
};

export const runStateSorter = (a: RunState, b: RunState): number => {
  return runStateSortValues[a] - runStateSortValues[b];
};

export const stringTimeSorter = (a: string, b: string): number => {
  const aTime = new Date(a).getTime();
  const bTime = new Date(b).getTime();
  return aTime - bTime;
};

export const taskStateSorter = (a: State, b: State): number => {
  // FIXME this is O(n) we can do it in constant time.
  // What is the right typescript way of doing it?
  const aValue = Object.values(RunState).includes(a as RunState) ?
    runStateSortValues[a as RunState] : commandStateSortValues[a as CommandState];
  const bValue = Object.values(RunState).includes(b as RunState) ?
    runStateSortValues[b as RunState] : commandStateSortValues[b as CommandState];
  return aValue - bValue;
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
