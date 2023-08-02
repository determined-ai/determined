/*
 * Hook to measure the width of a vertical scrollbar in px. May return 0 on mobile devices.
 * Based on this answer from StackOverflow: https://stackoverflow.com/a/55278118
 */
import { useEffect, useState } from 'react';

const useScrollbarWidth = (): number => {
  const [scrollbarWidth, setScrollbarWidth] = useState<number>(0);

  useEffect(() => {
    // Add temporary box to wrapper
    const scrollbox = document.createElement('div');

    // Make box scrollable
    scrollbox.style.overflow = 'scroll';

    // Append box to document
    document.body.appendChild(scrollbox);

    // Measure inner width of box
    setScrollbarWidth(scrollbox.offsetWidth - scrollbox.clientWidth);

    // Remove box
    document.body.removeChild(scrollbox);
  }, []);

  return scrollbarWidth;
};

export default useScrollbarWidth;
