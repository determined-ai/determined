import {
  CommandState, HpImportance, MetricName, MetricType, NullOrUndefined, Primitive, RunState, State,
} from 'types';

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
  [RunState.Deleting]: 7,
  [RunState.DeleteFailed]: 7,
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

/*
 * Sort numbers and strings with the following properties.
 *    - case insensitive
 *    - numbers come before string
 *    - place `null` and `undefined` at the end of numbers and strings
 */
export const alphaNumericSorter = (
  a: NullOrUndefined<string | number>,
  b: NullOrUndefined<string | number>,
): number => {
  // Handle undefined and null cases.
  if (a == null || b == null) return nullSorter(a, b);

  // Sort with English locale.
  return a.toString().localeCompare(b.toString(), 'en', { numeric: true });
};

export const booleanSorter = (a: NullOrUndefined<boolean>, b: NullOrUndefined<boolean>): number => {
  // Handle undefined and null cases.
  if (a == null || b == null) return nullSorter(a, b);

  // True values first.
  return (a === b) ? 0 : (a ? -1 : 1);
};

export const commandStateSorter = (a: CommandState, b: CommandState): number => {
  return commandStateSortValues[a] - commandStateSortValues[b];
};

/*
 * Sorts ISO 8601 datetime strings.
 * https://tc39.es/ecma262/#sec-date-time-string-format
 */
export const dateTimeStringSorter = (
  a: NullOrUndefined<string>,
  b: NullOrUndefined<string>,
): number => {
  // Handle undefined and null cases.
  if (a == null || b == null) return nullSorter(a, b);

  // Compare as date objects.
  const [ aTime, bTime ] = [ new Date(a).getTime(), new Date(b).getTime() ];
  if (aTime === bTime) return 0;
  return aTime < bTime ? -1 : 1;
};

export const hpImportanceSorter = (a: string, b: string, hpImportance: HpImportance): number => {
  const aValue = hpImportance[a];
  const bValue = hpImportance[b];
  if (aValue < bValue) return 1;
  if (aValue > bValue) return -1;
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
  return alphaNumericSorter(a.name, b.name);
};

/*
 * This also handles `undefined` and treats it equally as `null`.
 * NOTE: `undefined == null` is true (double equal sign not triple)
 */
export const nullSorter = (a: unknown, b: unknown): number => {
  if (a != null && b == null) return -1;
  if (a == null && b != null) return 1;
  return 0;
};

export const numericSorter = (a: NullOrUndefined<number>, b: NullOrUndefined<number>): number => {
  // Handle undefined and null cases.
  if (a == null || b == null) return nullSorter(a, b);

  // Sort by numeric type.
  if (a === b) return 0;
  return a < b ? -1 : 1;
};

export const primitiveSorter = (
  a: NullOrUndefined<Primitive>,
  b: NullOrUndefined<Primitive>,
): number => {
  // Handle undefined and null cases.
  if (a == null || b == null) return nullSorter(a, b);

  // Sort by primitive type.
  if (typeof a === 'boolean' && typeof b === 'boolean') return booleanSorter(a, b);
  if (typeof a === 'number' && typeof b === 'number') return numericSorter(a, b);
  if (typeof a === 'string' && typeof b === 'string') return alphaNumericSorter(a, b);
  return 0;
};

export const runStateSorter = (a: RunState, b: RunState): number => {
  return runStateSortValues[a] - runStateSortValues[b];
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
