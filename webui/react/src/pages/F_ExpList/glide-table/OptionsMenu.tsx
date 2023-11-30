import Button from 'hew/Button';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Toggle from 'hew/Toggle';
import { TypeOf } from 'io-ts';
import { useMemo } from 'react';

import { valueof } from 'utils/valueof';

import { TableViewMode } from './GlideTable';
import css from './OptionsMenu.module.scss';

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
  onTableViewModeChange?: (view: TableViewMode) => void;
  rowHeight: RowHeight;
  tableViewMode: TableViewMode;
}

export const OptionsMenu: React.FC<OptionProps> = ({
  rowHeight,
  tableViewMode,
  onRowHeightChange,
  onTableViewModeChange,
}) => {
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
      {
        children: [
          {
            icon: <Icon decorative name="scroll" />,
            key: 'scroll',
            label: (
              <div className={css.scrollSettingsRow}>
                <span>Infinite scroll</span>
                <Toggle checked={tableViewMode === 'scroll'} />
              </div>
            ),
            onClick: () => onTableViewModeChange?.(tableViewMode === 'scroll' ? 'paged' : 'scroll'),
          },
        ],
        label: 'Data',
        type: 'group',
      },
    ],
    [tableViewMode, onRowHeightChange, onTableViewModeChange],
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
