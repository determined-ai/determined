import Button from 'hew/Button';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import { TypeOf } from 'io-ts';
import { useMemo } from 'react';

import { valueof } from 'utils/valueof';

export const RowHeight = {
  EXTRA_TALL: 'EXTRA_TALL',
  MEDIUM: 'MEDIUM',
  SHORT: 'SHORT',
  TALL: 'TALL',
} as const;

export const ioRowHeight = valueof(RowHeight);

export type RowHeight = TypeOf<typeof ioRowHeight>;

export const rowHeightItems = [
  {
    icon: <Icon decorative name="row-small" />,
    label: 'Short',
    rowHeight: RowHeight.SHORT,
  },
  {
    icon: <Icon decorative name="row-medium" />,
    label: 'Medium',
    rowHeight: RowHeight.MEDIUM,
  },
  {
    icon: <Icon decorative name="row-large" />,
    label: 'Tall',
    rowHeight: RowHeight.TALL,
  },
  {
    icon: <Icon decorative name="row-xl" />,
    label: 'Extra Tall',
    rowHeight: RowHeight.EXTRA_TALL,
  },
];

interface OptionProps {
  onRowHeightChange?: (r: RowHeight) => void;
  rowHeight: RowHeight;
}

export const OptionsMenu: React.FC<OptionProps> = ({ rowHeight, onRowHeightChange }) => {
  const dropdownItems: MenuItem[] = useMemo(
    () => [
      {
        children: rowHeightItems.map(({ rowHeight, ...itemProps }) => ({
          ...itemProps,
          key: `rowHeight-${rowHeight}`,
          onClick: () => onRowHeightChange?.(rowHeight),
        })),
        label: 'Row height',
        type: 'group',
      },
    ],
    [onRowHeightChange],
  );
  const icon = (
    <span className="anticon">
      <Icon decorative name="options" />
    </span>
  );
  return (
    <Dropdown menu={dropdownItems} selectedKeys={[`rowHeight-${rowHeight}`]}>
      <Button icon={icon} tooltip="Options" />
    </Dropdown>
  );
};
