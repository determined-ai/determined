import ansiConverter from 'ansi-to-html';
import dayjs from 'dayjs';

type NullOrUndefined<T = undefined> = T | null | undefined;

const GLASBEY = [
  [0, 155, 222],
  [87, 43, 255],
  [158, 73, 209],
  [184, 125, 112],
  [0, 128, 90],
  [253, 229, 0],
  [149, 75, 119],
  [140, 213, 104],
  [114, 58, 62],
  [63, 65, 172],
  [102, 162, 214],
  [206, 105, 193],
  [94, 89, 106],
  [237, 172, 136],
  [106, 166, 160],
  [230, 170, 210],
  [99, 0, 136],
  [219, 253, 0],
  [25, 40, 104],
  [255, 66, 180],
  [197, 89, 14],
  [67, 135, 23],
  [0, 211, 145],
  [255, 90, 249],
  [102, 116, 91],
  [179, 174, 142],
  [140, 125, 156],
  [198, 0, 70],
  [46, 78, 108],
  [70, 109, 166],
  [115, 137, 158],
  [202, 175, 168],
  [167, 141, 206],
  [100, 254, 0],
  [0, 121, 146],
  [161, 99, 255],
  [216, 255, 245],
  [241, 140, 1],
  [160, 172, 20],
  [90, 46, 91],
  [158, 134, 137],
  [187, 204, 208],
  [197, 175, 212],
  [109, 221, 219],
  [244, 255, 208],
  [134, 101, 0],
  [99, 105, 0],
  [104, 65, 168],
  [197, 151, 45],
  [255, 116, 169],
  [94, 187, 39],
  [0, 183, 88],
  [167, 255, 203],
  [171, 122, 164],
  [148, 189, 255],
  [193, 226, 137],
  [255, 201, 15],
  [197, 0, 213],
  [138, 109, 99],
  [143, 133, 105],
  [83, 78, 75],
  [104, 96, 171],
  [213, 182, 122],
  [23, 90, 44],
  [37, 0, 154],
  [243, 209, 190],
  [104, 111, 138],
  [107, 165, 106],
  [104, 84, 134],
  [186, 205, 174],
  [127, 153, 136],
  [0, 220, 203],
  [145, 4, 155],
  [27, 188, 235],
  [210, 156, 235],
  [111, 0, 112],
  [50, 161, 177],
  [147, 108, 202],
  [164, 70, 65],
  [138, 140, 229],
  [0, 69, 213],
  [203, 139, 199],
  [151, 150, 183],
  [118, 32, 212],
  [204, 75, 114],
  [0, 78, 104],
  [56, 34, 104],
  [79, 86, 56],
  [171, 187, 111],
  [49, 58, 134],
  [152, 211, 165],
  [143, 175, 185],
  [223, 228, 216],
  [224, 0, 171],
  [219, 193, 203],
  [140, 223, 255],
  [77, 83, 227],
  [111, 105, 102],
  [28, 0, 255],
  [115, 45, 83],
  [108, 145, 78],
  [17, 109, 168],
  [38, 159, 255],
  [176, 163, 95],
  [87, 133, 200],
  [152, 89, 146],
  [255, 161, 163],
  [186, 186, 254],
  [136, 42, 37],
  [168, 230, 219],
  [167, 242, 151],
  [214, 148, 103],
  [64, 91, 186],
  [146, 93, 58],
  [47, 79, 54],
  [150, 124, 39],
  [155, 149, 138],
  [87, 180, 208],
  [100, 71, 0],
  [47, 93, 95],
  [65, 142, 142],
  [19, 63, 173],
  [60, 150, 106],
  [133, 61, 161],
  [186, 183, 191],
  [103, 198, 172],
  [207, 105, 101],
  [0, 176, 146],
  [218, 227, 44],
  [54, 111, 1],
  [83, 121, 255],
  [127, 129, 66],
  [0, 233, 79],
  [40, 84, 153],
  [0, 10, 93],
  [88, 0, 163],
  [0, 136, 12],
  [167, 131, 90],
  [251, 236, 255],
  [1, 105, 75],
  [212, 118, 136],
  [255, 199, 230],
  [218, 255, 165],
  [120, 111, 216],
  [75, 2, 223],
  [92, 104, 106],
  [162, 107, 120],
  [103, 128, 126],
  [134, 71, 90],
  [202, 0, 0],
  [43, 0, 124],
  [114, 255, 151],
  [225, 227, 182],
  [201, 83, 220],
  [52, 120, 119],
  [142, 190, 88],
];

const isBigInt = (data: unknown): data is bigint => typeof data === 'bigint';
const isBoolean = (data: unknown): data is boolean => typeof data === 'boolean';
const isMap = (data: unknown): data is Map<unknown, unknown> => data instanceof Map;
const isNullOrUndefined = (data: unknown): data is null | undefined => data == null;
export const isNumber = (data: unknown): data is number => typeof data === 'number';
const isPrimitive = (data: unknown): boolean =>
  isBigInt(data) ||
  isBoolean(data) ||
  isNullOrUndefined(data) ||
  isNumber(data) ||
  isString(data) ||
  isSymbol(data);
const isSet = (data: unknown): data is Set<unknown> => data instanceof Set;
const isString = (data: unknown): data is string => typeof data === 'string';
const isSymbol = (data: unknown): data is symbol => typeof data === 'symbol';

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

/*
 * This also handles `undefined` and treats it equally as `null`.
 * NOTE: `undefined == null` is true (double equal sign not triple)
 */
const nullSorter = (a: unknown, b: unknown): number => {
  if (a != null && b == null) return -1;
  if (a == null && b != null) return 1;
  return 0;
};

export const toHtmlId = (str: string): string => {
  return str
    .replace(/[\s_]/gi, '-')
    .replace(/[^a-z0-9-]/gi, '')
    .toLowerCase();
};

export const truncate = (str: string, maxLength = 20, suffix = '...'): string => {
  if (maxLength < suffix.length + 1) {
    maxLength = suffix.length + 1;
  }
  if (str.length <= maxLength) {
    return str;
  }
  return str.slice(0, maxLength - suffix.length) + suffix;
};

export const copyToClipboard = async (content: string): Promise<void> => {
  try {
    // This method is only available on https and localhost
    await navigator.clipboard.writeText(content);
  } catch (e) {
    throw new Error('Clipboard access on https and localhost only!');
  }
};

const LETTERS = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';
const CHARACTERS = `0123456789${LETTERS}`;
const DEFAULT_ALPHA_NUMERIC_LENGTH = 8;

export const generateAlphaNumeric = (
  length = DEFAULT_ALPHA_NUMERIC_LENGTH,
  chars = CHARACTERS,
): string => {
  let result = '';
  for (let i = length; i > 0; --i) {
    result += chars[Math.floor(Math.random() * chars.length)];
  }
  return result;
};

export const generateUUID = (): string => {
  return [
    generateAlphaNumeric(8),
    generateAlphaNumeric(4),
    generateAlphaNumeric(4),
    generateAlphaNumeric(4),
    generateAlphaNumeric(12),
  ].join('-');
};

// eslint-disable-next-line
export const clone = (data: any, deep = true): any => {
  if (isPrimitive(data)) return data;
  if (isMap(data)) return new Map(data);
  if (isSet(data)) return new Set(data);
  return deep ? JSON.parse(JSON.stringify(data)) : { ...data };
};

const DEFAULT_DATETIME_FORMAT = 'YYYY-MM-DD, HH:mm:ss';

export const formatDatetime = (
  datetime: string,
  options: { format?: string; inputUTC?: boolean; outputUTC?: boolean } = {},
): string => {
  const config = {
    format: DEFAULT_DATETIME_FORMAT,
    inputUTC: false,
    outputUTC: true,
    ...options,
  };
  // Strip out the timezone info if we want to force UTC input.
  const dateString = config.inputUTC ? stripTimezone(datetime) : datetime;

  // `dayjs.utc` respects timezone in the datetime string if available.
  let dayjsDate = dayjs.utc(dateString);

  // Prep the date as UTC or local time based on output UTC option.
  if (!config.outputUTC) dayjsDate = dayjsDate.local();

  // Return the formatted date based on provided format.
  return dayjsDate.format(config.format);
};

const stripTimezone = (datetime: string): string => {
  const timezoneRegex = /(Z|(-|\+)\d{2}:\d{2})$/;
  return datetime.replace(timezoneRegex, '');
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

export const numericSorter = (a: NullOrUndefined<number>, b: NullOrUndefined<number>): number => {
  // Handle undefined and null cases.
  if (a == null || b == null) return nullSorter(a, b);

  // Sort by numeric type.
  if (a === b) return 0;
  return a < b ? -1 : 1;
};

const converter = new ansiConverter({ newline: true });

export const ansiToHtml = (ansi: string): string => {
  const ansiWithHtml = ansi
    .replace(/(&|\u0026)/g, '&amp;')
    .replace(/(>|\u003e)/g, '&gt;')
    .replace(/(<|\u003c)/g, '&lt;')
    .replace(/('|\u0027)/g, '&apos;')
    .replace(/("|\u0022)/g, '&quot;');
  return converter.toHtml(ansiWithHtml);
};

/** titlecase a sentence */
export const capitalize = (str: string): string => {
  return str
    .split(/\s+/)
    .map((part) => capitalizeWord(part))
    .join(' ');
};

const capitalizeWord = (str: string): string => {
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
};

export const getCssVar = (name: string): string => {
  const varName = name.replace(/^(var\()?(.*?)\)?$/i, '$2');
  return window.getComputedStyle(document.body)?.getPropertyValue(varName);
};

export const glasbeyColor = (sequence: number): string => {
  const index = sequence % GLASBEY.length;
  const rgb = GLASBEY[index];
  return `rgb(${rgb[0]}, ${rgb[1]}, ${rgb[2]})`;
};

export const getTimeTickValues: uPlot.Axis.Values = (_self, rawValue) => {
  return rawValue.map((val) => dayjs.unix(val).format('hh:mm:ss.SSS').slice(0, -2));
};
