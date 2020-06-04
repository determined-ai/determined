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

interface ScrollOptions {
  behavior?: 'auto' | 'smooth';
  left?: number;
  top?: number;
}

type ResizeHandler = (entries: Element[]) => void;
type ScrollToFn = (options: ScrollOptions) => Promise<void>;
type ScrollHook = { scroll: ScrollInfo; scrollTo: ScrollToFn };

const SCROLL_EVENT = 'scroll';

export const useScroll = (ref: RefObject<HTMLElement>): ScrollHook => {
  const element = ref.current;
  const [ internalListener, setInternalListener ] = useState<EventListener | null>(null);
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

  const handleResize = useCallback(entries => {
    // Check to make sure the scroll element is being observed for resize.
    const elements = entries.map((entry: ResizeObserverEntry) => entry.target);
    if (!element || elements.indexOf(element) === -1) return;

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

  const scrollTo = useCallback((options: ScrollOptions): Promise<void> => {
    if (!element) return Promise.reject();

    // Clean up previous listener if applicable.
    if (internalListener) {
      element.removeEventListener(SCROLL_EVENT, internalListener);
      setInternalListener(null);
    }

    return new Promise(resolve => {
      const scrollListener: EventListener = event => {
        if (!event) return;
        const target = event.currentTarget as HTMLElement;
        if (target.scrollTop === options.top) {
          target.removeEventListener(SCROLL_EVENT, scrollListener);
          resolve();
        }
      };

      setInternalListener(scrollListener);
      element.addEventListener(SCROLL_EVENT, scrollListener);
      element.scroll(options);
    });
  }, [ element, internalListener ]);

  useEffect(() => {
    if (!element) return;

    const resizeObserver = new ResizeObserver(handleResize);
    resizeObserver.observe(element);
    element.addEventListener(SCROLL_EVENT, handleScroll);

    return (): void => {
      resizeObserver.unobserve(element);
      element.removeEventListener(SCROLL_EVENT, handleScroll);
      if (internalListener) element.removeEventListener(SCROLL_EVENT, internalListener);
    };
  }, [ element, handleResize, handleScroll, internalListener ]);

  return { scroll: scrollInfo, scrollTo };
};

export default useScroll;
