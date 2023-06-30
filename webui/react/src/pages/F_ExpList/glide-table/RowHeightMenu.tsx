import { useMemo } from 'react';

import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';

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
      <Icon key={'icon_0'} name="row-small" title="small row" />,
      <Icon key={'icon_1'} name="row-medium" title="medium row" />,
      <Icon key={'icon_2'} name="row-large" title="large row" />,
      <Icon key={'icon_3'} name="row-xl" title="extra large row" />,
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
      <Icon name="options" title="row height" />
    </span>
  );
  return (
    <Dropdown menu={dropdownItems} selectedKeys={[`rowHeight-${rowHeight}`]}>
      <Button icon={icon} tooltip="Row height" />
    </Dropdown>
  );
};
