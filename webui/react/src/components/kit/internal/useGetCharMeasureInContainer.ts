import { RefObject, useMemo } from 'react';

export interface CharMeasure {
  height: number;
  width: number;
}

const useGetCharMeasureInContainer = (container: RefObject<HTMLElement>): CharMeasure => {
  const containerInner = container.current;

  const elem = document.createElement('div');
  elem.style.display = 'inline';
  elem.style.opacity = '0';
  elem.style.position = 'fixed';
  elem.style.top = '0';
  elem.style.width = 'auto';
  elem.style.visibility = 'hidden';
  elem.textContent = 'W';
  containerInner?.appendChild?.(elem);

  return useMemo(() => {
    if (!containerInner) {
      return {
        height: 0,
        width: 0,
      };
    }

    const charRect = elem.getBoundingClientRect();

    return {
      height: charRect.height,
      width: charRect.width,
    };
  }, [containerInner, elem]);
};

export default useGetCharMeasureInContainer;
