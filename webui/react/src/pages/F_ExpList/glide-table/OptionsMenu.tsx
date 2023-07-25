import { useMemo } from 'react';

import Button from 'components/kit/Button';
import Dropdown, { MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import Toggle from 'components/kit/Toggle';

import { ExpListView, RowHeight } from '../F_ExperimentList.settings';

import css from './OptionsMenu.module.scss';

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
          key: `rowHeight-${rowHeight}`,
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
                <Toggle checked={expListView === 'scroll'} />
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
