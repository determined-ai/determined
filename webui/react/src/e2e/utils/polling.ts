/**
 * Creates a function that alternates between two functions.
 * The first time the created function is called, it runs the first function.
 * Subsequent calls run the second function and then the first function again.
 * @param {() => Promise<void>} repeat The function to repeat.
 * @param {() => Promise<void>} fallback The function to reset state if we repeat again.
 * @returns A function that runs a fallback in between calls of repeat.
 */
export function repeatWithFallback(repeat: () => Promise<void>, fallback: () => Promise<void>): () => Promise<void> {
  let isFirstFunction = true;

  return async () => {
    if (isFirstFunction) {
      isFirstFunction = false;
    } else {
      await fallback();
    }
    await repeat();
  };
}
