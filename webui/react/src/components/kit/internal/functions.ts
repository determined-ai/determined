import ansiConverter from 'ansi-to-html';
import dayjs from 'dayjs';
import utc from 'dayjs/plugin/utc';
dayjs.extend(utc);

import { Metric, NullOrUndefined } from 'components/kit/internal/types';

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
export const isObject = (data: unknown): boolean => {
  return typeof data === 'object' && !Array.isArray(data) && !isSet(data) && data !== null;
};
const isPrimitive = (data: unknown): boolean =>
  isBigInt(data) ||
  isBoolean(data) ||
  isNullOrUndefined(data) ||
  isNumber(data) ||
  isString(data) ||
  isSymbol(data);
const isSet = (data: unknown): data is Set<unknown> => data instanceof Set;
export const isString = (data: unknown): data is string => typeof data === 'string';
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
  return window
    .getComputedStyle(document.getElementsByClassName('ui-provider')[0])
    ?.getPropertyValue(varName);
};

export const glasbeyColor = (sequence: number): string => {
  const index = sequence % GLASBEY.length;
  const rgb = GLASBEY[index];
  return `rgb(${rgb[0]}, ${rgb[1]}, ${rgb[2]})`;
};

export const getTimeTickValues: uPlot.Axis.Values = (_self, rawValue) => {
  return rawValue.map((val) => dayjs.unix(val).format('hh:mm:ss.SSS').slice(0, -2));
};

const DEFAULT_PRECISION = 6;
export const humanReadableNumber = (num: number, precision = DEFAULT_PRECISION): string => {
  const stringNum = num.toString();
  let content: string = stringNum;

  if (isNaN(num)) {
    content = 'NaN';
  } else if (!Number.isFinite(num)) {
    content = `${num < 0 ? '-' : ''}Infinity`;
  } else if (!Number.isInteger(num)) {
    content = num.toFixed(Math.max(precision, 0));

    const absoluteNum = Math.abs(num);
    if (absoluteNum < 0.01 || absoluteNum > 999) {
      content = num.toExponential(Math.max(precision, 0));
    }
  }

  return content;
};

// credits: https://gist.github.com/Izhaki/834a9d37d1ad34c6179b6a16e670b526
export const findInsertionIndex = (
  sortedArray: number[],
  value: number,
  compareFn: (a: number, b: number) => number = (a, b) => a - b,
): number => {
  // empty array
  if (sortedArray.length === 0) return 0;

  // value beyond current sortedArray range
  if (compareFn(value, sortedArray[sortedArray.length - 1]) >= 0) return sortedArray.length;

  const getMidPoint = (start: number, end: number): number => Math.floor((end - start) / 2) + start;

  let iEnd = sortedArray.length - 1;
  let iStart = 0;

  let iMiddle = getMidPoint(iStart, iEnd);

  // binary search
  while (iStart < iEnd) {
    const comparison = compareFn(value, sortedArray[iMiddle]);

    // found match
    if (comparison === 0) return iMiddle;

    if (comparison < 0) {
      // target is lower in array, move the index halfway down
      iEnd = iMiddle;
    } else {
      // target is higher in array, move the index halfway up
      iStart = iMiddle + 1;
    }
    iMiddle = getMidPoint(iStart, iEnd);
  }

  return iMiddle;
};

export function distance(x0: number, y0: number, x1: number, y1: number): number {
  return Math.sqrt((x1 - x0) ** 2 + (y1 - y0) ** 2);
}

/*
 * h - hue between 0 and 360
 * s - saturation between 0.0 and 1.0
 * l - lightness between 0.0 and 1.0
 */
interface HslColor {
  h: number;
  l: number;
  s: number;
}

/*
 * r - red between 0 and 255
 * g - green between 0 and 255
 * b - blue between 0 and 255
 * a - alpha between 0.0 and 1.0
 */
interface RgbaColor {
  a?: number;
  b: number;
  g: number;
  r: number;
}
const hexRegex = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i;
const rgbaRegex = /^rgba?\(\s*?(\d+)\s*?,\s*?(\d+)\s*?,\s*?(\d+)\s*?(,\s*?([\d.]+)\s*?)?\)$/i;

export const rgba2str = (rgba: RgbaColor): string => {
  if (rgba.a != null) {
    return `rgba(${rgba.r}, ${rgba.g}, ${rgba.b}, ${rgba.a})`;
  }
  return `rgb(${rgba.r}, ${rgba.g}, ${rgba.b})`;
};

export const rgbaFromGradient = (
  rgba0: RgbaColor,
  rgba1: RgbaColor,
  percent: number,
): RgbaColor => {
  const r = Math.round((rgba1.r - rgba0.r) * percent + rgba0.r);
  const g = Math.round((rgba1.g - rgba0.g) * percent + rgba0.g);
  const b = Math.round((rgba1.b - rgba0.b) * percent + rgba0.b);

  if (rgba0.a != null && rgba1.a != null) {
    const a = (rgba1.a - rgba0.a) * percent + rgba0.a;
    return { a, b, g, r };
  }

  return { b, g, r };
};

export const hex2rgb = (hex: string): RgbaColor => {
  const rgb = { b: 0, g: 0, r: 0 };
  const result = hexRegex.exec(hex);

  if (result && result.length > 3) {
    rgb.r = parseInt(result[1], 16);
    rgb.g = parseInt(result[2], 16);
    rgb.b = parseInt(result[3], 16);
  }

  return rgb;
};

export const str2rgba = (str: string): RgbaColor => {
  if (hexRegex.test(str)) return hex2rgb(str);

  const regex = rgbaRegex;
  const result = regex.exec(str);
  if (result && result.length > 3) {
    const rgba = { a: 1.0, b: 0, g: 0, r: 0 };
    rgba.r = parseInt(result[1]);
    rgba.g = parseInt(result[2]);
    rgba.b = parseInt(result[3]);
    if (result.length > 5 && result[5] != null) rgba.a = parseFloat(result[5]);
    return rgba;
  }

  return { a: 0.0, b: 0, g: 0, r: 0 };
};

export const hex2hsl = (hex: string): HslColor => {
  return rgba2hsl(hex2rgb(hex));
};

export const rgba2hsl = (rgba: RgbaColor): HslColor => {
  const r = rgba.r / 255;
  const g = rgba.g / 255;
  const b = rgba.b / 255;
  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  const avg = (max + min) / 2;
  const hsl: HslColor = { h: Math.round(Math.random() * 6), l: 0.5, s: 0.5 };

  hsl.h = hsl.s = hsl.l = avg;

  if (max === min) {
    hsl.h = hsl.s = 0; // achromatic
  } else {
    const d = max - min;
    hsl.s = hsl.l > 0.5 ? d / (2 - max - min) : d / (max + min);
    switch (max) {
      case r:
        hsl.h = (g - b) / d + (g < b ? 6 : 0);
        break;
      case g:
        hsl.h = (b - r) / d + 2;
        break;
      case b:
        hsl.h = (r - g) / d + 4;
        break;
    }
  }

  hsl.h = Math.round((360 * hsl.h) / 6);
  hsl.s = Math.round(hsl.s * 100);
  hsl.l = Math.round(hsl.l * 100);

  return hsl;
};

export const hsl2str = (hsl: HslColor): string => {
  return `hsl(${hsl.h}, ${hsl.s}%, ${hsl.l}%)`;
};

const METRIC_KEY_DELIMITER = '.';

export const metricToStr = (metric: Metric, truncateLimit = 30): string => {
  /**
   * TODO - also see `src/components/MetricBadgeTag.tsx'
   * Metric group may sometimes end up being `undefined` when an old metric setting
   * is restored and the UI attempts to use it. Adding a safeguard for now.
   * Better approach of hunting down all the places it can be stored as a setting
   * and validating it upon loading and discarding it if invalid.
   */
  const label = !metric.group
    ? metric.name
    : [metric.group, metric.name].join(METRIC_KEY_DELIMITER);
  return label.length > truncateLimit ? label.substring(0, truncateLimit) + '...' : label;
};
