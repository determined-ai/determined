import React, { useCallback, useState } from 'react';

import Button from 'components/kit/Button';
import Dropdown, { DropdownEvent, MenuItem } from 'components/kit/Dropdown';
import Icon from 'components/kit/Icon';
import Tooltip from 'components/kit/Tooltip';
import { isMouseEvent } from 'utils/routes';

import css from './PageHeaderFoldable.module.scss';

export interface Option {
  disabled?: boolean;
  icon?: React.ReactNode;
  isLoading?: boolean;
  key: string;
  label: React.ReactNode;
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
    <Tooltip content={option.tooltip}>
      <span>{option.label}</span>
    </Tooltip>
  ) : (
    <span>{option.label}</span>
  );
};

const PageHeaderFoldable: React.FC<Props> = ({ foldableContent, leftContent, options }: Props) => {
  const [isExpanded, setIsExpanded] = useState(false);

  const dropdownClasses = [css.optionsDropdown];
  if (options?.length === 1) dropdownClasses.push(css.optionsDropdownOneChild);
  if (options?.length === 2) dropdownClasses.push(css.optionsDropdownTwoChild);
  if (options?.length === 3) dropdownClasses.push(css.optionsDropdownThreeChild);

  const menu: MenuItem[] = (options ?? []).map((option) => ({
    className: css.optionsDropdownItem,
    disabled: option.disabled || !option.onClick,
    key: option.key,
    label: renderOptionLabel(option),
  }));

  const handleDropdown = useCallback(
    (key: string, e: DropdownEvent) => {
      const option = options?.find((option) => option.key === key);
      if (isMouseEvent(e)) option?.onClick?.(e);
    },
    [options],
  );

  return (
    <div className={css.base}>
      <div className={css.header}>
        <div className={css.left}>{leftContent}</div>
        <div className={css.options}>
          {foldableContent && (
            <Button
              icon={
                <Icon
                  name={isExpanded ? 'arrow-up' : 'arrow-down'}
                  showTooltip
                  size="tiny"
                  title="Toggle expansion"
                />
              }
              type="text"
              onClick={() => setIsExpanded((prev) => !prev)}
            />
          )}
          <div className={css.optionsButtons}>
            {options?.slice(0, 3).map((option) => (
              <div className={css.optionsMainButton} key={option.key}>
                <Button
                  disabled={option.disabled || !option.onClick}
                  icon={option?.icon}
                  key={option.key}
                  loading={option.isLoading}
                  onClick={option.onClick}>
                  {renderOptionLabel(option)}
                </Button>
              </div>
            ))}
          </div>
          {menu.length !== 0 && (
            <Dropdown menu={menu} placement="bottomRight" onClick={handleDropdown}>
              <div className={dropdownClasses.join(' ')}>
                <Button icon={<Icon name="overflow-vertical" title="Action menu" />} />
              </div>
            </Dropdown>
          )}
        </div>
      </div>
      {foldableContent && isExpanded && <div className={css.foldable}>{foldableContent}</div>}
    </div>
  );
};

export default PageHeaderFoldable;
