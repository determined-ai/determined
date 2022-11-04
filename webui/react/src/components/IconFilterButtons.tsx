import { Space } from 'antd';
import React, { useCallback } from 'react';

import IconButton from './IconButton';
import css from './IconFilterButtons.module.scss';

interface FilterButton {
  active: boolean;
  icon: string;
  id: string;
  label: string;
}

interface Props {
  buttons: FilterButton[];
  onClick?: (id: string, e?: React.MouseEvent) => void;
}

const FilterButtons: React.FC<Props> = ({ buttons, onClick }: Props) => {
  const handleClick = useCallback(
    (id: string) => {
      return (e: React.MouseEvent) => onClick?.(id, e);
    },
    [onClick],
  );

  return (
    <Space className={css.base}>
      {buttons.map((button) => (
        <IconButton
          className={button.active ? css.active : css.inactive}
          icon={button.icon}
          key={button.id}
          label={button.label}
          tooltipPlacement="top"
          onClick={handleClick(button.id)}
        />
      ))}
    </Space>
  );
};

export default FilterButtons;
