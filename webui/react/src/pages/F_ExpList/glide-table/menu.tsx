import { CheckOutlined, SmileOutlined } from '@ant-design/icons';
import { Menu, MenuProps } from 'antd';
import { ItemType } from 'antd/es/menu/hooks/useItems';
import React, { MutableRefObject, useEffect, useRef } from 'react';

import { V1ProjectColumn } from 'services/api-ts-sdk';

import { DirectionType, optionsByColumnType, Sort } from './MultiSortMenu';

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

export const sortMenuItemsForColumn = (
  column: V1ProjectColumn,
  sorts: Sort[],
  onSortChange: (sorts: Sort[]) => void,
): ItemType[] =>
  optionsByColumnType[column.type].map((option) => {
    const curSort = sorts.find((s) => s.column === column.column);
    const isSortMatch = curSort && curSort.direction === option.value;
    return {
      icon: isSortMatch ? <CheckOutlined /> : <div />,
      key: option.value,
      label: `Sort ${option.label}`,
      onClick: () => {
        let newSort: Sort[];
        if (isSortMatch) {
          newSort = sorts.filter((s) => s.column !== column.column);
        } else if (curSort) {
          newSort = sorts.map((s) =>
            s.column !== column.column
              ? s
              : {
                  ...s,
                  direction: option.value as DirectionType,
                },
          );
        } else {
          newSort = [{ column: column.column, direction: option.value as DirectionType }];
        }
        onSortChange(newSort);
      },
    };
  });

export const placeholderMenuItems: ItemType[] = [
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
