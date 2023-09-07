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

const useResize = (): ResizeHook => {
  const elementRef = useRef(document.body);
  const [, setObserver] = useState<ResizeObserver>();
  const [resizeInfo, setResizeInfo] = useState<SizeInfo>({ ...DEFAULT_SIZE });

  const measureRef = useCallback((node: HTMLElement) => {
    if (node) elementRef.current = node;

    setObserver((prev) => {
      if (prev) prev.unobserve(elementRef.current);

      const handleResize: ResizeObserverCallback = (entries: ResizeObserverEntry[]) => {
        // Check to make sure the ref container is being observed for resize.
        const elements = entries.map((entry: ResizeObserverEntry) => entry.target);
        if (!elementRef.current || elements.indexOf(elementRef.current) === -1) return;

        const rect = elementRef.current.getBoundingClientRect();
        setResizeInfo(rect);
      };
      const resizeObserver = new ResizeObserver(handleResize);
      resizeObserver.observe(elementRef.current);

      return resizeObserver;
    });

    const rect = elementRef.current.getBoundingClientRect();
    setResizeInfo(rect);
  }, []);

  // Default resize target element to be document.body.
  useEffect(() => {
    measureRef(document.body);
  }, [measureRef]);

  return { refCallback: measureRef, refObject: elementRef, size: resizeInfo };
};

export default useResize;
