import { Menu, MenuProps } from 'antd';
import React, { MutableRefObject, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router';

import { paths } from 'routes/utils';

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

export interface TableContextMenuProps extends MenuProps {
  open: boolean;
  rowKey: number;
  handleClose: () => void;
  x: number;
  y: number;
}

export const TableContextMenu: React.FC<TableContextMenuProps> = ({
  x,
  y,
  open,
  rowKey,
  handleClose,
}) => {
  const containerRef = useRef(null);
  useOutsideClickHandler(containerRef, handleClose);

  const navigate = useNavigate();

  return (
    <div
      ref={containerRef}
      style={{
        border: 'solid 1px gold',
        display: !open ? 'none' : undefined,
        left: x,
        position: 'fixed',
        top: y,
        width: 200,
      }}>
      <Menu
        items={[
          {
            disabled: false,
            key: '1',
            label: 'Visit',
            onClick: () => navigate(paths.experimentDetails(rowKey)),
          },
        ]}
      />
    </div>
  );
};
