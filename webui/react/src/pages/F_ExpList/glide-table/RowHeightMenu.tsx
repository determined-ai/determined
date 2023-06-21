import { useMemo } from 'react';

import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';

import { RowHeight } from '../F_ExperimentList.settings';

import { RowHeight as RowHeightIcon } from './icons';

interface RowHeightMenuProps {
  rowHeight: RowHeight;
  onRowHeightChange: (r: RowHeight) => void;
}

const rowHeightCopy: Record<RowHeight, string> = {
  [RowHeight.SHORT]: 'Short',
  [RowHeight.MEDIUM]: 'Medium',
  [RowHeight.TALL]: 'Tall',
  [RowHeight.EXTRA_TALL]: 'Extra Tall',
};

export const RowHeightMenu: React.FC<RowHeightMenuProps> = ({ rowHeight, onRowHeightChange }) => {
  const dropdownItems: MenuItem[] = useMemo(
    () => [
      {
        children: Object.entries(rowHeightCopy).map(([rowHeight, label]) => ({
          key: `rowHeight-${rowHeight}`,
          label,
          onClick: () => onRowHeightChange(rowHeight as RowHeight),
        })),
        label: 'Row height',
        type: 'group',
      },
    ],
    [onRowHeightChange],
  );
  const icon = (
    <span className="anticon">
      <RowHeightIcon />
    </span>
  );
  return (
    <Dropdown menu={dropdownItems} selectedKeys={[`rowHeight-${rowHeight}`]}>
      <Button icon={icon} tooltip="Row height" />
    </Dropdown>
  );
};
