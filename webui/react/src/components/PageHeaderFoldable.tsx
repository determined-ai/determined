import { Button, Dropdown, Menu, Tooltip } from 'antd';
import React, { CSSProperties, useState } from 'react';

import Icon from 'components/Icon';

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
  style?: CSSProperties;
}

const renderOptionLabel =
  (option: Option): React.ReactNode => {
    return option.tooltip
      ? <Tooltip title={option.tooltip}><span>{option.label}</span></Tooltip>
      : <span>{option.label}</span>;
  };

const PageHeaderFoldable: React.FC<Props> = (
  { foldableContent, leftContent, options, style }: Props,
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
            onClick={(e) => opt.onClick && opt.onClick(e.domEvent)}
          >{renderOptionLabel(opt)}</Menu.Item>
        ))}
      </Menu>
    );
  }

  return (
    <>
      <div className={css.base} style={style}>

        <div className={css.left}>
          {leftContent}
        </div>

        {foldableContent && (
          <div className={css.toggle} onClick={() => setIsExpanded(!isExpanded)}>
            <Icon name={isExpanded ? 'arrow-up' : 'arrow-down'} />
          </div>
        )}

        <div className={css.options}>
          {options && options.slice(0, 3).map((opt, i) => (
            <Button
              className={css.optionsMainButton}
              disabled={!opt.onClick}
              ghost={i !== 0}
              icon={opt?.icon}
              key={opt.key}
              loading={opt.isLoading}
              onClick={opt.onClick}
            >{renderOptionLabel(opt)}</Button>
          ))}
          {dropdownOptions && (
            <Dropdown arrow overlay={dropdownOptions} placement="bottomRight">
              <Button
                className={dropdownClasses.join(' ')}
                ghost={true}
                icon={<Icon name="overflow-vertical" />}
              />
            </Dropdown>
          )}
        </div>

        {foldableContent && isExpanded && (
          <div className={css.bottom}>
            {foldableContent}
          </div>
        )}

      </div>
    </>
  );
};

export default PageHeaderFoldable;
