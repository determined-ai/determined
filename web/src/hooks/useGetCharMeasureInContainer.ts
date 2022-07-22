import { RefObject, useMemo } from 'react';

export interface CharMeasure {
  height: number;
  width: number;
}

const useGetCharMeasureInContainer = (container: RefObject<HTMLDivElement>): CharMeasure => {
  const containerInner = container.current;

  return useMemo(() => {
    if (!containerInner) {
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
    elem.textContent = 'W';
    containerInner.appendChild(elem);

    const charRect = elem.getBoundingClientRect();

    elem.remove();

    return {
      height: charRect.height,
      width: charRect.width,
    };
  }, [ containerInner ]);
};

export default useGetCharMeasureInContainer;
