// credits: https://stackoverflow.com/a/55533058
export const sumArrays = (...arrays: number[][]): number[] => {
  const n = arrays.reduce((max, xs) => Math.max(max, xs?.length), 0);
  const result = Array.from({ length: n });
  return result.map((_, i) => arrays.map(xs => xs[i] || 0).reduce((sum, x) => sum + x, 0));
};
