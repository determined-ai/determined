/*
 * Scrolling hook to detect scroll events on a target element.
 * Based on a discussion found below:
 * https://gist.github.com/joshuacerbito/ea318a6a7ca4336e9fadb9ae5bbb87f4
 */
import { RefObject, useCallback, useEffect, useState } from 'react';

interface ScrollInfo {
  dx: number;
  dy: number;
  scrollHeight: number;
  scrollLeft: number;
  scrollTop: number;
  scrollWidth: number;
  viewHeight: number;
  viewWidth: number;
}

const SCROLL_EVENT = 'scroll';

export const useScroll = (ref: RefObject<HTMLElement>): ScrollInfo => {
  const element = ref.current;
  const [ scroll, setScroll ] = useState<ScrollInfo>({
    dx: 0,
    dy: 0,
    scrollHeight: element?.scrollHeight || 0,
    scrollLeft: element?.scrollLeft || 0,
    scrollTop: element?.scrollTop || 0,
    scrollWidth: element?.scrollWidth || 0,
    viewHeight: element?.clientHeight || 0,
    viewWidth: element?.clientWidth || 0,
  });

  const listener = useCallback(() => {
    if (!element) return;
    setScroll(prev => ({
      dx: element?.scrollLeft - prev.scrollLeft,
      dy: element?.scrollTop - prev.scrollTop,
      scrollHeight: element?.scrollHeight,
      scrollLeft: element?.scrollLeft,
      scrollTop: element?.scrollTop,
      scrollWidth: element?.scrollWidth,
      viewHeight: element?.clientHeight,
      viewWidth: element?.clientWidth,
    }));
  }, [ element ]);

  useEffect(() => {
    if (!element) return;
    element.addEventListener(SCROLL_EVENT, listener);
    return (): void => element.removeEventListener(SCROLL_EVENT, listener);
  }, [ element, listener ]);

  return scroll;
};

export default useScroll;
