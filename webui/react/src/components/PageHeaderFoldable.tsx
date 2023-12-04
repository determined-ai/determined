import Button from 'hew/Button';
import Dropdown, { DropdownEvent, MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Tooltip from 'hew/Tooltip';
import React, { useCallback, useState } from 'react';

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
  content?: React.ReactNode;
}

interface HeaderOption {
  content: React.ReactNode;
  menuOption: Option;
}
export interface MenuOption {
  disabled?: boolean;
  key: string;
  label: React.ReactNode;
  onClick?: (ev: React.MouseEvent) => void;
  content: React.ReactNode;
  tooltip?: string;
}

interface Props {
  foldableContent?: React.ReactNode;
  leftContent: React.ReactNode;
  options?: HeaderOption[];
}

export const renderOptionLabel = (option: Option): React.ReactNode => {
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
    disabled: option.menuOption.disabled || !option.menuOption.onClick,
    key: option.menuOption.key,
    label: renderOptionLabel(option.menuOption),
  }));

  const handleDropdown = useCallback(
    (key: string, e: DropdownEvent) => {
      const option = options?.find((option) => option.menuOption.key === key);
      if (isMouseEvent(e)) option?.menuOption.onClick?.(e);
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
              <div className={css.optionsMainButton} key={option.menuOption.key}>
                {option.content}
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
