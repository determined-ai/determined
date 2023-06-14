import { useMemo } from 'react';

import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import OptionsIcon from 'components/kit/svgIcons/OptionsIcon';
import RowIcon from 'components/kit/svgIcons/RowIcon';

import { RowHeight } from '../F_ExperimentList.settings';

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
  const getIcon = (index: number) => {
    const icons = [
      <RowIcon key={'icon_0'} size="small" />,
      <RowIcon key={'icon_1'} />,
      <RowIcon key={'icon_2'} size="large" />,
      <RowIcon key={'icon_3'} size="xl" />,
    ];

    return icons[index];
  };
  const dropdownItems: MenuItem[] = useMemo(
    () => [
      {
        children: Object.entries(rowHeightCopy).map(([rowHeight, label], idx) => ({
          icon: getIcon(idx),
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
      <OptionsIcon />
    </span>
  );
  return (
    <Dropdown menu={dropdownItems} selectedKeys={[`rowHeight-${rowHeight}`]}>
      <Button icon={icon} tooltip="Row height" />
    </Dropdown>
  );
};
