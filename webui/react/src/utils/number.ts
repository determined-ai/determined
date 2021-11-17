import { isString } from './data';

const PERCENT_REGEX = /^(\d+\.?\d*|\.\d+)%$/;

export const findFactorOfNumber = (n: number): number[] => {
  const abs = Math.abs(n);
  const factorsAsc = [];
  const factorsDesc = [];

  for (let i = 1; i <= Math.floor(Math.sqrt(abs)); i++) {
    if (abs % i !== 0) continue;
    factorsAsc.push(i);

    if (abs / i === i) continue;
    factorsDesc.push(abs / i);
  }

  return factorsAsc.concat(factorsDesc.reverse());
};

export const isPercent = (data: unknown): boolean => {
  if (!isString(data)) return false;
  return PERCENT_REGEX.test(data);
};

export const percent = (n: number, decimals = 1): number => {
  const normalized = n < 0 ? 0 : (n > 1 ? 1 : (n || 0));
  const factor = Math.pow(10, decimals);
  return Math.round(normalized * 100 * factor) / factor;
};

export const percentToFloat = (percent: unknown): number => {
  if (isPercent(percent)) {
    const matches = (percent as string).match(PERCENT_REGEX);
    if (matches?.length === 2) return parseFloat(matches[1]) / 100;
  }
  return 1.0;
};

export const roundToPrecision = (n: number, precision = 6): number => {
  const factor = 10 ** precision;
  return Math.round((n + Number.EPSILON) * factor) / factor;
};

export const clamp = (val: number, min: number, max: number):number => {
  return Math.max(Math.min(val, max), min);
};
