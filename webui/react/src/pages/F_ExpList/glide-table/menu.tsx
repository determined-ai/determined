import { MenuProps } from 'antd';
import { ItemType } from 'antd/es/menu/hooks/useItems';
import React, { MutableRefObject, useEffect, useRef } from 'react';

import Dropdown, { MenuItem } from 'components/kit/Dropdown';

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
const isMenuItem = (val: ItemType): val is MenuItem =>
  val === null || !!val?.key || ('type' in val && val.type === 'divider');

export const TableActionMenu: React.FC<TableActionMenuProps> = ({
  x,
  y,
  open,
  handleClose,
  items,
}) => {
  const divRef = useRef<HTMLDivElement | null>(null);
  useOutsideClickHandler(divRef, handleClose);
  return (
    <Dropdown menu={items?.filter(isMenuItem)} open={open} placement="bottomLeft">
      <div
        ref={divRef}
        style={{
          left: x,
          position: 'fixed',
          top: y,
        }}
      />
    </Dropdown>
  );
};
