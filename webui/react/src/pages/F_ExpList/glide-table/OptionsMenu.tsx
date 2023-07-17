import { Switch } from 'antd';
import { useMemo } from 'react';

import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';

import { ExpListView, RowHeight } from '../F_ExperimentList.settings';

import css from './OptionsMenu.module.scss';

const rowHeightItems = [
  {
    icon: <Icon decorative name="row-small" />,
    key: 'rowHeight-SHORT',
    label: 'Short',
    rowHeight: RowHeight.SHORT,
  },
  {
    icon: <Icon decorative name="row-medium" />,
    key: 'rowHeight-MEDIUM',
    label: 'Medium',
    rowHeight: RowHeight.MEDIUM,
  },
  {
    icon: <Icon decorative name="row-large" />,
    key: 'rowHeight-TALL',
    label: 'Tall',
    rowHeight: RowHeight.TALL,
  },
  {
    icon: <Icon decorative name="row-xl" />,
    key: 'rowHeight-EXTRA_TALL',
    label: 'Extra Tall',
    rowHeight: RowHeight.EXTRA_TALL,
  },
];

interface OptionProps {
  expListView: ExpListView;
  onRowHeightChange: (r: RowHeight) => void;
  rowHeight: RowHeight;
  setExpListView: (v: ExpListView) => void;
}

export const OptionsMenu: React.FC<OptionProps> = ({
  rowHeight,
  onRowHeightChange,
  expListView,
  setExpListView,
}) => {
  const dropdownItems: MenuItem[] = useMemo(
    () => [
      {
        children: rowHeightItems.map(({ rowHeight, ...itemProps }) => ({
          ...itemProps,
          onClick: () => onRowHeightChange(rowHeight),
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
                <Switch checked={expListView === 'scroll'} size="small" />
              </div>
            ),
            onClick: () => setExpListView(expListView === 'scroll' ? 'paged' : 'scroll'),
          },
        ],
        label: 'Data',
        type: 'group',
      },
    ],
    [onRowHeightChange, expListView, setExpListView],
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
