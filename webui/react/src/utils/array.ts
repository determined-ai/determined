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

// credits: https://stackoverflow.com/a/55533058
export const sumArrays = (...arrays: number[][]): number[] => {
  const n = arrays.reduce((max, xs) => Math.max(max, xs?.length), 0);
  const result = Array.from({ length: n });
  return result.map((_, i) => arrays.map(xs => xs[i] || 0).reduce((sum, x) => sum + x, 0));
};
