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

const RESIZE_EVENT = 'resize';
const SCROLL_EVENT = 'scroll';

export const useScroll = (ref: RefObject<HTMLElement>): [ ScrollInfo, () => void ] => {
  const element = ref.current;
  const [ scrollInfo, setScrollInfo ] = useState<ScrollInfo>({
    dx: 0,
    dy: 0,
    scrollHeight: element?.scrollHeight || 0,
    scrollLeft: element?.scrollLeft || 0,
    scrollTop: element?.scrollTop || 0,
    scrollWidth: element?.scrollWidth || 0,
    viewHeight: element?.clientHeight || 0,
    viewWidth: element?.clientWidth || 0,
  });

  const handleResize = useCallback(() => {
    if (!element) return;
    setScrollInfo(prevScrollInfo => ({
      ...prevScrollInfo,
      scrollHeight: element.scrollHeight,
      scrollWidth: element.scrollWidth,
      viewHeight: element.clientHeight,
      viewWidth: element.clientWidth,
    }));
  }, [ element ]);

  const handleScroll = useCallback(() => {
    if (!element) return;
    setScrollInfo(prevScrollInfo => ({
      ...prevScrollInfo,
      dx: element.scrollLeft - prevScrollInfo.scrollLeft,
      dy: element.scrollTop - prevScrollInfo.scrollTop,
      scrollLeft: element.scrollLeft,
      scrollTop: element.scrollTop,
    }));
  }, [ element ]);

  useEffect(() => {
    if (!element) return;
    element.addEventListener(RESIZE_EVENT, handleResize);
    element.addEventListener(SCROLL_EVENT, handleScroll);
    return (): void => {
      element.removeEventListener(RESIZE_EVENT, handleResize);
      element.removeEventListener(SCROLL_EVENT, handleScroll);
    };
  }, [ element, handleResize, handleScroll ]);

  return [ scrollInfo, handleResize ];
};

export default useScroll;
