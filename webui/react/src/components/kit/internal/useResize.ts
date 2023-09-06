import { RefCallback, RefObject, useCallback, useEffect, useRef, useState } from 'react';

interface ResizeInfo {
  height: number;
  width: number;
  x: number;
  y: number;
}

interface ResizeHook {
  elementRef: RefObject<HTMLElement>;
  ref: RefCallback<HTMLElement>;
  size: ResizeInfo;
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
  const [resizeInfo, setResizeInfo] = useState<ResizeInfo>({ ...DEFAULT_SIZE });

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

  return { elementRef, ref: measureRef, size: resizeInfo };
};

export default useResize;
