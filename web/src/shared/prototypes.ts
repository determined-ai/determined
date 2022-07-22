export {};

/*
 * Generally adding to top level JS prototypes is a bad idea,
 * due to potentially polluting top level namespaces.
 * However, Array is an exception because it is unlikely to
 * happen and provides a lot of value.
 */

/*
 * Note: When requesting an item from an Array with an
 * out of range index, JS will return `undefined`.
 */

Array.prototype.first = function() {
  return this[0];
};

Array.prototype.last = function() {
  return this[this.length - 1];
};

Array.prototype.random = function() {
  const index = Math.floor((Math.random() * this.length));
  return this[index];
};

function swap<T>(arr: T[], i: number, j: number): T[] {
  const temp = arr[i];
  arr[i] = arr[j];
  arr[j] = temp;
  return arr;
}

function partition<T>(arr: T[], low: number, high: number, compareFn: (a: T, b: T) => number) {
  let q = low;
  let i;
  for (i = low; i < high; i++) {
    if (compareFn(arr[i], arr[high]) === -1) {
      swap(arr, i, q);
      q += 1;
    }
  }
  swap(arr, i, q);
  return q;
}

function quickSort<T>(arr: T[], low: number, high: number, compareFn: (a: T, b: T) => number) {
  if (low < high) {
    const pivot = partition(arr, low, high, compareFn);
    quickSort(arr, low, pivot - 1, compareFn);
    quickSort(arr, pivot + 1, high, compareFn);
    return arr;
  }
  return [];
}

/*
 * Native Array.prototype.sort ignores `undefined` values inside the array,
 * so they always end up at the end of the list regardless of the sorting function.
 * Array.prototype.sortAll will sort `undefined` also and treat them the same as
 * how `null` values are treated.
 */
Array.prototype.sortAll = function(compareFn) {
  return quickSort(this, 0, this.length - 1, compareFn);
};

Storage.prototype.keys = function() {
  return Object.keys(this);
};
