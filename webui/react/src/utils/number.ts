export const percent = (n: number, decimals = 1): number => {
  if (n < 0 || n > 1) throw new Error('input out of range');
  const factor = Math.pow(10, decimals);
  return Math.round(n * 100 * factor) / factor;
};
