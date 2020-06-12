import { useEffect, useRef } from 'react';

const usePrevious = <T>(value: T, defaultValue: T): T => {
  const ref = useRef<T>(defaultValue);

  useEffect(() => {
    ref.current = value;
  });

  return ref.current;
};

export default usePrevious;
