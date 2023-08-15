import { NullOrUndefined, Primitive, SemanticVersion, Workspace } from 'types';

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
  return a === b ? 0 : a ? -1 : 1;
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
  const [aTime, bTime] = [new Date(a).getTime(), new Date(b).getTime()];
  if (aTime === bTime) return 0;
  return aTime < bTime ? -1 : 1;
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

export const workspaceSorter = (a: Workspace, b: Workspace): number => {
  // Keep `Uncategorized` at the very top.
  if (a.id === 1) return -1;
  if (b.id === 1) return 1;

  // Sort the remainder by workspace name.
  return alphaNumericSorter(a.name, b.name);
};

/** return true if a semantic version a is older than b */
export const semVerIsOlder = (a: SemanticVersion, b: SemanticVersion): boolean => {
  return (
    a.major < b.major ||
    (a.major === b.major && a.minor < b.minor) ||
    (a.major === b.major && a.minor === b.minor && a.patch < b.patch)
  );
};

/** sort a list of versions from latest to oldest. */
export const sortVersions = (versions: SemanticVersion[]): SemanticVersion[] => {
  return versions.sort((a, b) => (semVerIsOlder(a, b) ? 1 : -1));
};
