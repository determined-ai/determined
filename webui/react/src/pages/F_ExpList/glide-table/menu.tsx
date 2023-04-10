import { SmileOutlined } from '@ant-design/icons';
import { Menu, MenuProps } from 'antd';
import React, { MutableRefObject, useEffect, useRef } from 'react';

function useOutsideClickHandler(
  // eslint-disable-next-line
  ref: MutableRefObject<any>,
  handler: () => void,
  shouldSkip: boolean,
) {
  useEffect(() => {
    if (shouldSkip) {
      return;
    }
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
  }, [ref, handler, shouldSkip]);
}

export interface TableActionMenuProps extends MenuProps {
  x: number;
  y: number;
  open: boolean;
  handleClick?: () => void;
  handleClose: () => void;
  isContextMenu?: boolean;
}

export const TableActionMenu: React.FC<TableActionMenuProps> = ({
  x,
  y,
  open,
  handleClick,
  handleClose,
  items,
  isContextMenu = false,
}) => {
  const containerRef = useRef(null);
  useOutsideClickHandler(containerRef, handleClose, isContextMenu);

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
      }}
      onClick={handleClick}>
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

export const contextMenuItems: MenuProps['items'] = [
  {
    disabled: false,
    key: '1',
    label: 'Pin row',
  },
];

export const pinnedContextMenuItems: MenuProps['items'] = [
  {
    disabled: false,
    key: '1',
    label: 'Unpin row',
  },
];
