import { SmileOutlined } from '@ant-design/icons';
import { Menu, MenuProps } from 'antd';
import React, { MutableRefObject, useEffect, useRef } from 'react';

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
  const containerRef = useRef(null);
  useOutsideClickHandler(containerRef, handleClose);

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
      <Menu items={items} />
    </div>
  );
};

export const placeholderMenuItems: MenuProps['items'] = [
  {
    disabled: false,
    icon: <SmileOutlined />,
    key: '1',
    label: 'Menu Placeholder',
  },
  {
    disabled: false,
    icon: <SmileOutlined />,
    key: '2',
    label: 'Other Menu Thing',
  },
];
