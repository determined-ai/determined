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
