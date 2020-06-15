import { useEffect, useRef } from 'react';

/*
 * This hook takes advantage of useRef hook of retaining an
 * older state to preserve and return the previous state
 * of a useState hook.
 */
const usePrevious = <T>(value: T, defaultValue: T): T => {
  const ref = useRef<T>(defaultValue);

  useEffect(() => {
    ref.current = value;
  });

  return ref.current;
};

export default usePrevious;
