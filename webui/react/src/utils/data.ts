import { CommandState, RunState, State } from 'types';

export const isMap = <T>(data: T): boolean => data instanceof Map;
export const isNumber = <T>(data: T): boolean => typeof data === 'number';
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

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const getPath = <T>(obj: Record<string, any>, path: string): T | undefined => {
  // Reassigns to obj[key] on each array.every iteration
  let value = obj || {};
  return path.split('.').every(key => ((value = value[key]) !== undefined)) ?
    value as T : undefined;
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

export const alphanumericSorter = (a: string|number, b: string|number): number => {
  return a.toString().localeCompare(b.toString(), 'en', { numeric: true });
};

export const commandStateSorter = (a: CommandState, b: CommandState): number => {
  return commandStateSortValues[a] - commandStateSortValues[b];
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
