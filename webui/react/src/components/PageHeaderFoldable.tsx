import { Dropdown } from 'antd';
import type { DropdownProps, MenuProps } from 'antd';
import React, { useState } from 'react';

import Button from 'components/kit/Button';
import Tooltip from 'components/kit/Tooltip';
import Icon from 'shared/components/Icon/Icon';
import { isMouseEvent } from 'shared/utils/routes';

import css from './PageHeaderFoldable.module.scss';

export interface Option {
  disabled?: boolean;
  icon?: React.ReactNode;
  isLoading?: boolean;
  key: string;
  label: string;
  onClick?: (ev: React.MouseEvent) => void;
  tooltip?: string;
}

interface Props {
  foldableContent?: React.ReactNode;
  leftContent: React.ReactNode;
  options?: Option[];
}

const renderOptionLabel = (option: Option): React.ReactNode => {
  return option.tooltip ? (
    <Tooltip title={option.tooltip}>
      <span>{option.label}</span>
    </Tooltip>
  ) : (
    <span>{option.label}</span>
  );
};

const PageHeaderFoldable: React.FC<Props> = ({ foldableContent, leftContent, options }: Props) => {
  const [isExpanded, setIsExpanded] = useState(false);

  let dropdownOptions: DropdownProps['menu'] = {};
  if (options && options.length > 0) {
    const onItemClick: MenuProps['onClick'] = (e) => {
      const opt = options.find((opt) => opt.key === e.key) as Option;
      if (isMouseEvent(e.domEvent)) {
        opt.onClick?.(e.domEvent);
      }
    };

    const menuItems: MenuProps['items'] = options.map((opt) => ({
      className: css.optionsDropdownItem,
      disabled: opt.disabled || !opt.onClick,
      key: opt.key,
      label: renderOptionLabel(opt),
    }));

    dropdownOptions = { items: menuItems, onClick: onItemClick };
  }

  return (
    <div className={css.base}>
      <div className={css.header}>
        <div className={css.left}>{leftContent}</div>
        <div className={css.options}>
          {foldableContent && (
            <Tooltip title="Toggle">
              <Button type="text" onClick={() => setIsExpanded((prev) => !prev)}>
                <Icon name={isExpanded ? 'arrow-up' : 'arrow-down'} size="tiny" />
              </Button>
            </Tooltip>
          )}
          <div className={css.optionsButtons}>
            {options?.slice(0, 3).map((option) => (
              <Button
                disabled={option.disabled || !option.onClick}
                ghost
                icon={option?.icon}
                key={option.key}
                loading={option.isLoading}
                onClick={option.onClick}>
                {renderOptionLabel(option)}
              </Button>
            ))}
          </div>
          {dropdownOptions && (
            <Dropdown menu={dropdownOptions} placement="bottomRight" trigger={['click']}>
              <Button ghost icon={<Icon name="overflow-vertical" />} />
            </Dropdown>
          )}
        </div>
      </div>
      {foldableContent && isExpanded && <div className={css.foldable}>{foldableContent}</div>}
    </div>
  );
};

export default PageHeaderFoldable;
