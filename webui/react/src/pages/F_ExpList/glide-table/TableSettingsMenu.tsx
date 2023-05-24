import { useMemo } from 'react';

import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';

import { RowHeight } from '../F_ExperimentList.settings';

import { ComfyHeight, CompactHeight, DefaultHeight } from './icons';

interface TableSettingsMenuProps {
  rowHeight: RowHeight;
  onRowHeightChange: (r: RowHeight) => void;
}

/* eslint-disable react/jsx-key  */
const rowHeightCopy: Record<RowHeight, [string, React.ReactNode]> = {
  [RowHeight.COMFY]: ['Comfy', <ComfyHeight />],
  [RowHeight.DEFAULT]: ['Default', <DefaultHeight />],
  [RowHeight.COMPACT]: ['Compact', <CompactHeight />],
};
/* eslint-enable react/jsx-key  */

const rowHeightDropdownOptions = (onRowHeightChange: (r: RowHeight) => void): MenuItem => ({
  children: Object.entries(rowHeightCopy).map(([rowHeight, [label, icon]]) => ({
    icon: <span className="anticon">{icon}</span>,
    key: `rowHeight-${rowHeight}`,
    label,
    onClick: () => onRowHeightChange(rowHeight as RowHeight),
  })),
  label: 'Row height',
  type: 'group',
});

export const TableSettingsMenu: React.FC<TableSettingsMenuProps> = ({
  rowHeight,
  onRowHeightChange,
}) => {
  const dropdownItems: MenuItem[] = useMemo(
    () => [rowHeightDropdownOptions(onRowHeightChange)],
    [onRowHeightChange],
  );
  return (
    <Dropdown menu={dropdownItems} selectedKeys={[`rowHeight-${rowHeight}`]}>
      <Button>Options</Button>
    </Dropdown>
  );
};
