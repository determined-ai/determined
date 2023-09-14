import { RefObject, useMemo } from 'react';

import useResize from 'components/kit/internal/useResize';

export interface CharMeasure {
  height: number;
  width: number;
}

const useGetCharMeasureInContainer = (container: RefObject<HTMLElement>): CharMeasure => {
  const { size } = useResize(container);

  return useMemo(() => {
    if (!container.current) {
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [size, container]);
};

export default useGetCharMeasureInContainer;
