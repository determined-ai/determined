/*
 * Scrolling hook to detect scroll events on a target element.
 * Based on a discussion found below:
 * https://gist.github.com/joshuacerbito/ea318a6a7ca4336e9fadb9ae5bbb87f4
 */
import { RefObject, useCallback, useEffect, useState } from 'react';
import smoothScroll from 'smoothscroll-polyfill';

// Kick off the polyfill.
smoothScroll.polyfill();

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

export const defaultScrollInfo = {
  dx: 0,
  dy: 0,
  scrollHeight: 0,
  scrollLeft: 0,
  scrollTop: 0,
  scrollWidth: 0,
  viewHeight: 0,
  viewWidth: 0,
};

export const useScroll = (ref: RefObject<HTMLElement>): ScrollInfo => {
  const element = ref.current;
  const [ scrollInfo, setScrollInfo ] = useState<ScrollInfo>({
    ...defaultScrollInfo,
    scrollHeight: element?.scrollHeight || 0,
    scrollLeft: element?.scrollLeft || 0,
    scrollTop: element?.scrollTop || 0,
    scrollWidth: element?.scrollWidth || 0,
    viewHeight: element?.clientHeight || 0,
    viewWidth: element?.clientWidth || 0,
  });

  const handleResize = useCallback(entries => {
    // Check to make sure the scroll element is being observed for resize.
    const elements = entries.map((entry: ResizeObserverEntry) => entry.target);
    if (!element || elements.indexOf(element) === -1) return;

    setScrollInfo(prevScrollInfo => ({
      ...prevScrollInfo,
      dx: element.scrollLeft - prevScrollInfo.scrollLeft,
      dy: element.scrollTop - prevScrollInfo.scrollTop,
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
      scrollHeight: element.scrollHeight,
      scrollLeft: element.scrollLeft,
      scrollTop: element.scrollTop,
      scrollWidth: element.scrollWidth,
    }));
  }, [ element ]);

  useEffect(() => {
    if (!element) return;

    const resizeObserver = new ResizeObserver(handleResize);
    resizeObserver.observe(element);
    element.addEventListener(SCROLL_EVENT, handleScroll);

    return (): void => {
      resizeObserver.unobserve(element);
      element.removeEventListener(SCROLL_EVENT, handleScroll);
    };
  }, [ element, handleResize, handleScroll ]);

  return scrollInfo;
};

export default useScroll;
