import React, { useEffect, useMemo, useRef, useState } from 'react';
import { throttle } from 'throttle-debounce';

import useMobile from 'hooks/useMobile';
import useResize from 'hooks/useResize';

import css from './SplitPane.module.scss';

interface Props {
  children: [React.ReactElement, React.ReactElement];
  initialWidth?: number;
  minimumWidths?: [number, number];
  onChange?: (width: number) => void;
  open?: boolean;
}

const SplitPane: React.FC<Props> = ({
  children,
  initialWidth = 400,
  minimumWidths = [200, 200],
  onChange,
  open = true,
}: Props) => {
  const [isDragging, setIsDragging] = useState(false);
  const [width, setWidth] = useState(initialWidth);
  const container = useRef<HTMLDivElement>(null);
  const handle = useRef<HTMLDivElement>(null);
  const containerDimensions = useResize(container);
  const isMobile = useMobile();

  const throttledOnChange = useMemo(
    () => onChange && throttle(8, onChange, { noTrailing: true }),
    [onChange],
  );

  useEffect(() => setWidth(initialWidth), [initialWidth]);

  useEffect(() => {
    const handleDragStart = (e: MouseEvent) => {
      if (e.button !== 0) return;
      e.preventDefault();
      setIsDragging(true);
    };
    const handleRef = handle.current;
    handleRef?.addEventListener('mousedown', handleDragStart);

    return () => handleRef?.removeEventListener('mousedown', handleDragStart);
  }, []);

  useEffect(() => {
    if (!isDragging) return;
    const handleDrag = (e: MouseEvent) => {
      e.preventDefault();

      // Get x-coordinate of pointer relative to container
      const pointerRelativeXpos = e.clientX - containerDimensions.x;

      // * 8px is the left/right spacing between .handle and its inner pseudo-element
      const newWidth = Math.round(
        Math.min(
          Math.max(minimumWidths[0], pointerRelativeXpos - 8),
          containerDimensions.width - minimumWidths[1],
        ),
      );

      // Resize box A
      setWidth(newWidth);
      throttledOnChange?.(newWidth);
    };
    document.addEventListener('mousemove', handleDrag);

    return () => document.removeEventListener('mousemove', handleDrag);
  }, [
    containerDimensions.width,
    containerDimensions.x,
    throttledOnChange,
    isDragging,
    minimumWidths,
  ]);

  useEffect(() => {
    if (!isDragging) return;
    const handleDragStop = (e: MouseEvent) => {
      if (e.button !== 0) return;
      // Turn off dragging flag when user mouse is up
      setIsDragging(false);
      throttledOnChange?.cancel();
      onChange?.(width);
    };
    document.addEventListener('mouseup', handleDragStop);

    return () => document.removeEventListener('mouseup', handleDragStop);
  }, [width, isDragging, onChange, throttledOnChange]);

  const classnames = [css.base];
  if (open) classnames.push(css.open);

  return (
    <div className={classnames.join(' ')} ref={container}>
      <div style={{ display: open && isMobile ? 'none' : 'initial', width: open ? width : '100%' }}>
        {children[0]}
      </div>
      {
        <div
          className={css.handle}
          ref={handle}
          style={{ display: isMobile ? 'none' : 'initial' }}
        />
      }
      <div className={css.rightBox}>{children[1]}</div>
    </div>
  );
};

export default SplitPane;
