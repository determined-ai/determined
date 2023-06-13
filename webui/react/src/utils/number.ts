import { isString } from './data';

const PERCENT_REGEX = /^(\d+\.?\d*|\.\d+)%$/;
const DEFAULT_PRECISION = 6;

export const clamp = (val: number, min: number, max: number): number => {
  return Math.max(Math.min(val, max), min);
};

export const findFactorOfNumber = (n: number): number[] => {
  const abs = Math.abs(n);
  const factorsAsc: number[] = [];
  const factorsDesc: number[] = [];

  for (let i = 1; i <= Math.floor(Math.sqrt(abs)); i++) {
    if (abs % i !== 0) continue;
    factorsAsc.push(i);

    if (abs / i === i) continue;
    factorsDesc.push(abs / i);
  }

  return factorsAsc.concat(factorsDesc.reverse());
};

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

export const nearestCardinalNumber = (num: number): string => {
  const suffixes = ['k', 'M', 'B', 'T'];

  if (isNaN(num)) {
    return 'NaN';
  } else if (!Number.isFinite(num)) {
    return `${num < 0 ? '-' : ''}Infinity`;
  } else if (Math.abs(num) < 1000) {
    return Math.round(num).toString();
  }

  let tempNum = Math.round(num);
  let iters = 0;
  while (Math.abs(tempNum) >= 1000) {
    tempNum /= 1000;
    iters++;
  }

  const suffix = [];
  if (iters <= suffixes.length) {
    suffix.push(suffixes[iters - 1]);
  } else {
    while (iters > suffixes.length) {
      suffix.push(suffixes.at(-1));
      iters -= suffixes.length;
    }
    if (iters > 0) {
      suffix.unshift(suffixes[iters - 1]);
    }
  }

  if (Math.abs(tempNum) < 10) {
    return (
      ((Math.sign(tempNum) * Math.floor(Math.sign(tempNum) * tempNum * 10)) / 10).toFixed(1) +
      suffix.join('')
    );
  }
  return Math.sign(tempNum) * Math.floor(Math.sign(tempNum) * tempNum) + suffix.join('');
};

export const isPercent = (data: unknown): boolean => {
  if (!isString(data)) return false;
  return PERCENT_REGEX.test(data);
};

export const percent = (n: number, decimals = 1): number => {
  const normalized = clamp(n || 0, 0, 1);
  const factor = Math.pow(10, decimals);
  return Math.round(normalized * 100 * factor) / factor;
};

export const percentToFloat = (percent: unknown): number => {
  if (isPercent(percent)) return parseFloat(percent as string) / 100;
  return 1.0;
};

export const roundToPrecision = (n: number, precision = 6): number => {
  const factor = 10 ** precision;
  return Math.round((n + Number.EPSILON) * factor) / factor;
};
