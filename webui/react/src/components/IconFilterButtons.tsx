import { Tooltip } from 'antd';
import React, { useCallback } from 'react';

import Icon from './Icon';
import css from './IconFilterButtons.module.scss';

interface FilterButton {
  active: boolean;
  icon: string;
  id: string;
  label: string;
}

interface Props {
  buttons: FilterButton[];
  onClick?: (id: string) => void;
}

const FilterButtons: React.FC<Props> = ({ buttons, onClick }: Props) => {
  const handleClick = useCallback((id: string): (() => void) => {
    return (): void => {
      if (onClick) onClick(id);
    };
  }, [ onClick ]);

  return (
    <div className={css.base}>
      <div>
        <div className={css.buttons}>
          {buttons.map(button => {
            const buttonClasses = [ css.button ];
            if (button.active) buttonClasses.push(css.active);
            return (
              <Tooltip key={button.id} placement="top" title={button.label}>
                <button
                  aria-label={button.label}
                  className={buttonClasses.join(' ')}
                  tabIndex={0}
                  onClick={handleClick(button.id)}>
                  <Icon name={button.icon} />
                </button>
              </Tooltip>
            );
          })}
        </div>
      </div>
    </div>
  );
};

export default FilterButtons;
