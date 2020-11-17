import { RefObject, useEffect, useState } from 'react';

interface ResizeInfo {
  height: number;
  width: number;
  x: number;
  y: number;
}

const defaultResizeInfo = {
  height: 0,
  width: 0,
  x: 0,
  y: 0,
};

export const DEFAULT_RESIZE_THROTTLE_TIME = 500;

const useResize = (ref?: RefObject<HTMLElement>): ResizeInfo => {
  const element = ref ? ref.current : document.body;
  const [ resizeInfo, setResizeInfo ] = useState<ResizeInfo>(defaultResizeInfo);

  useEffect(() => {
    if (!element) return;

    const handleResize: ResizeObserverCallback = entries => {
      // Check to make sure the ref container is being observed for resize.
      const elements = entries.map((entry: ResizeObserverEntry) => entry.target);
      if (!element || elements.indexOf(element) === -1) return;

      const rect = element.getBoundingClientRect();
      setResizeInfo(rect);
    };
    const resizeObserver = new ResizeObserver(handleResize);
    resizeObserver.observe(element);

    return (): void => resizeObserver.unobserve(element);
  }, [ element ]);

  return resizeInfo;
};

export default useResize;
