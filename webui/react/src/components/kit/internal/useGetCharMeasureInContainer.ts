import { RefObject, useEffect, useMemo, useState } from 'react';

export interface CharMeasure {
  height: number;
  width: number;
}

const useEffectInEvent = (set?: () => void) => {
  useEffect(() => {
    set?.();
    if (set) {
      window.addEventListener('resize', set);
      return () => window.removeEventListener('resize', set);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);
};

const useGetCharMeasureInContainer = (container: RefObject<HTMLElement>): CharMeasure => {
  const [rect, setRect] = useState<DOMRect>();
  const set = () => setRect(container.current?.getBoundingClientRect());
  useEffectInEvent(set);

  return useMemo(() => {
    if (!rect) {
      return {
        height: 0,
        width: 0,
      };
    }

    const elem = document.createElement('div');
    elem.style.display = 'inline';
    elem.style.opacity = '0';
    elem.style.position = 'fixed';
    elem.style.top = '0';
    elem.style.width = 'auto';
    elem.style.visibility = 'hidden';
    elem.textContent = 'W';
    container.current?.appendChild?.(elem);

    const charRect = elem.getBoundingClientRect();
    elem.remove();

    return {
      height: charRect.height,
      width: charRect.width,
    };
  }, [rect, container]);
};

export default useGetCharMeasureInContainer;
