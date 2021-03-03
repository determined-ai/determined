export const percent = (n: number, decimals = 1): number => {
  const normalized = n < 0 ? 0 : (n > 1 ? 1 : (n || 0));
  const factor = Math.pow(10, decimals);
  return Math.round(normalized * 100 * factor) / factor;
};

export const roundToPrecision = (n: number, precision = 6): number => {
  const factor = 10 ** precision;
  return Math.round((n + Number.EPSILON) * factor) / factor;
};
