import React, { useEffect, useRef, useState } from 'react';

import css from './SplitPane.module.scss';

interface Props {
  children: [React.ReactElement, React.ReactElement];
  initialWidth?: number;
  onChange?: (width: number) => void;
  open?: boolean;
}

const SplitPane: React.FC<Props> = ({ children, initialWidth, onChange, open = true }: Props) => {
  const [isDragging, setIsDragging] = useState(false);
  const [width, setWidth] = useState(initialWidth ?? 400);
  const container = useRef<HTMLDivElement>(null);
  const handle = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const c = (e: MouseEvent) => {
      e.preventDefault();
      setIsDragging(true);
    };
    const handleRef = handle.current;
    handleRef?.addEventListener('mousedown', c);

    return () => handleRef?.removeEventListener('mousedown', c);
  }, []);

  useEffect(() => {
    if (!isDragging) return;
    const c = (e: MouseEvent) => {
      if (!container.current) return;

      e.preventDefault();

      // Get offset
      const containerOffsetLeft = container.current.getBoundingClientRect().left;

      // Get x-coordinate of pointer relative to container
      const pointerRelativeXpos = e.clientX - containerOffsetLeft;

      // Arbitrary minimum width set on box A, otherwise its inner content will collapse to width of 0
      const boxAminWidth = 200;

      // * 8px is the left/right spacing between .handle and its inner pseudo-element
      const newWidth = Math.max(boxAminWidth, pointerRelativeXpos - 8);

      // Resize box A
      setWidth(newWidth);
    };
    document.addEventListener('mousemove', c);

    return () => document.removeEventListener('mousemove', c);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isDragging]);

  useEffect(() => {
    const c = () => {
      // Turn off dragging flag when user mouse is up
      setIsDragging(false);
      onChange?.(width);
    };

    document.addEventListener('mouseup', c);

    return () => document.removeEventListener('mouseup', c);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleClassnames = [css.handle];
  const rightClassnames = [css.rightBox];

  if (open) rightClassnames.push(css.open) && handleClassnames.push(css.open);

  return (
    <div className={css.base} ref={container}>
      <div className={css.leftBox} style={{ width: open ? width : '100%' }}>
        {children?.[0]}
      </div>
      <div className={handleClassnames.join(' ')} ref={handle} />
      <div className={rightClassnames.join(' ')}>{children?.[1]}</div>
    </div>
  );
};

export default SplitPane;
