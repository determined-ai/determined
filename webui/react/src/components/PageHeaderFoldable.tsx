import { Button, Dropdown, Menu, Space, Tooltip } from 'antd';
import React, { useState } from 'react';

import Icon from 'components/Icon';

import { isMouseEvent } from '../shared/utils/routes';

import IconButton from './IconButton';
import css from './PageHeaderFoldable.module.scss';

export interface Option {
  icon?: React.ReactNode,
  isLoading?: boolean,
  key: string;
  label: string;
  onClick?: (ev: React.MouseEvent) => void;
  tooltip?: string;
}

interface Props {
  foldableContent?: React.ReactNode,
  leftContent: React.ReactNode,
  options?: Option[],
}

const renderOptionLabel =
  (option: Option): React.ReactNode => {
    return option.tooltip
      ? <Tooltip title={option.tooltip}><span>{option.label}</span></Tooltip>
      : <span>{option.label}</span>;
  };

const PageHeaderFoldable: React.FC<Props> = (
  { foldableContent, leftContent, options }: Props,
) => {
  const [ isExpanded, setIsExpanded ] = useState(false);

  const dropdownClasses = [ css.optionsDropdown ];
  let dropdownOptions = null;
  if (options && options.length > 0) {
    if (options.length === 1) dropdownClasses.push(css.optionsDropdownOneChild);
    if (options.length === 2) dropdownClasses.push(css.optionsDropdownTwoChild);
    if (options.length === 3) dropdownClasses.push(css.optionsDropdownThreeChild);
    dropdownOptions = (
      <Menu>
        {options.map(opt => (
          <Menu.Item
            className={css.optionsDropdownItem}
            disabled={!opt.onClick}
            key={opt.key}
            onClick={(e) => isMouseEvent(e.domEvent) && opt.onClick && opt.onClick(e.domEvent)}>
            {renderOptionLabel(opt)}
          </Menu.Item>
        ))}
      </Menu>
    );
  }

  return (
    <div className={css.base}>
      <div className={css.header}>
        <div className={css.left}>
          {leftContent}
        </div>
        {foldableContent && (
          <IconButton
            icon={isExpanded ? 'arrow-up' : 'arrow-down'}
            iconSize="tiny"
            label="Toggle"
            type="text"
            onClick={() => setIsExpanded(prev => !prev)}
          />
        )}
        <Space className={css.options}>
          {options?.slice(0, 3).map(option => (
            <Button
              disabled={!option.onClick}
              icon={option?.icon}
              key={option.key}
              loading={option.isLoading}
              onClick={option.onClick}>{renderOptionLabel(option)}
            </Button>
          ))}
          {dropdownOptions && (
            <Dropdown overlay={dropdownOptions} placement="bottomRight" trigger={[ 'click' ]}>
              <Button
                className={dropdownClasses.join(' ')}
                ghost
                icon={<Icon name="overflow-vertical" />}
              />
            </Dropdown>
          )}
        </Space>
      </div>
      {foldableContent && isExpanded && <div className={css.foldable}>{foldableContent}</div>}
    </div>
  );
};

export default PageHeaderFoldable;
