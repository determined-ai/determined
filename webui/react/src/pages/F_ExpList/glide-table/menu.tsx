import { Menu, MenuProps } from 'antd';
import React, { MutableRefObject, useEffect, useRef } from 'react';

import useResize from 'hooks/useResize';

// eslint-disable-next-line
function useOutsideClickHandler(ref: MutableRefObject<any>, handler: () => void) {
  useEffect(() => {
    /**
     * Alert if clicked on outside of element
     */
    function handleClickOutside(event: Event) {
      if (ref.current && !ref.current.contains(event.target)) {
        handler();
      }
    }
    // Bind the event listener
    document.addEventListener('mouseup', handleClickOutside);
    return () => {
      // Unbind the event listener on clean up
      document.removeEventListener('mouseup', handleClickOutside);
    };
  }, [ref, handler]);
}

export interface TableActionMenuProps extends MenuProps {
  x: number;
  y: number;
  open: boolean;
  handleClose: () => void;
}

export const TableActionMenu: React.FC<TableActionMenuProps> = ({
  x,
  y,
  open,
  handleClose,
  items,
}) => {
  const menuWidth = 220;
  const containerRef = useRef(null);
  useOutsideClickHandler(containerRef, handleClose);
  const { width } = useResize();

  return (
    <div
      ref={containerRef}
      style={{
        border: 'solid 1px gray',
        display: !open ? 'none' : undefined,
        left: width - x < menuWidth ? width - menuWidth : x,
        position: 'fixed',
        top: y,
        width: menuWidth,
        zIndex: 100,
      }}>
      <Menu items={items} selectable={false} />
    </div>
  );
};
