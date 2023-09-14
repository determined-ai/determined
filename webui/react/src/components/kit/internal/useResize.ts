import { RefCallback, RefObject, useCallback, useEffect, useRef, useState } from 'react';

interface SizeInfo {
  height: number;
  width: number;
  x: number;
  y: number;
}

interface ResizeHook {
  refObject: RefObject<HTMLElement>;
  refCallback: RefCallback<HTMLElement>;
  size: SizeInfo;
}

const DEFAULT_SIZE = {
  height: 0,
  width: 0,
  x: 0,
  y: 0,
};

const useResize = (ref?: RefObject<HTMLElement>): ResizeHook => {
  const elementRef = useRef<HTMLElement>(ref?.current ?? document.body);
  const isMeasured = useRef(false);
  const observer = useRef<ResizeObserver>();
  const [resizeInfo, setResizeInfo] = useState<SizeInfo>({ ...DEFAULT_SIZE });

  const measureRef = useCallback((node: HTMLElement) => {
    isMeasured.current = true;

    // Tear down previous resize observer.
    observer.current?.unobserve(elementRef.current);

    if (node) elementRef.current = node;

    // Set up resize observer.
    const handleResize: ResizeObserverCallback = (entries: ResizeObserverEntry[]) => {
      // Check to make sure the ref container is being observed for resize.
      const elements = entries.map((entry: ResizeObserverEntry) => entry.target);
      if (!elementRef.current || elements.indexOf(elementRef.current) === -1) return;

      const rect = elementRef.current.getBoundingClientRect();
      setResizeInfo(rect);
    };
    observer.current = new ResizeObserver(handleResize);
    observer.current?.observe(elementRef.current);

    const rect = elementRef.current.getBoundingClientRect();
    setResizeInfo(rect);
  }, []);

  // If the `refCallback` is not applied, run measure against `document.body`
  useEffect(() => {
    if (!isMeasured.current) measureRef(document.body);
  }, [measureRef]);

  // When hook unmounts clean up observer if applicable.
  useEffect(() => {
    return () => {
      observer.current?.unobserve(elementRef.current);
      observer.current = undefined;
    };
  }, []);

  return { refCallback: measureRef, refObject: elementRef, size: resizeInfo };
};

export default useResize;
