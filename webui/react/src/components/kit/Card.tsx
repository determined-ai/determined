import { Dropdown, MenuProps } from 'antd';
import React from 'react';

import Icon from 'shared/components/Icon';

import Button from './Button';
import css from './Card.module.scss';

interface CardProps {
  actionMenu?: MenuProps;
  children?: React.ReactNode;
  disabled?: boolean;
  height?: number;
  onClick?: () => void;
  width?: number;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const Card: React.FC<CardProps> = ({
  actionMenu,
  children,
  disabled = false,
  onClick,
  height = 184,
  width = 184,
}: CardProps) => {
  return (
    <div
      className={css.base}
      style={{ minHeight: `${height}px`, width: `${width}px` }}
      tabIndex={onClick ? 0 : -1}
      onClick={onClick}>
      {actionMenu && (
        <div className={css.action}>
          <Dropdown
            disabled={disabled}
            menu={actionMenu}
            placement="bottomRight"
            trigger={['click']}>
            <Button disabled={disabled} type="text" onClick={stopPropagation}>
              <Icon name="overflow-horizontal" />
            </Button>
          </Dropdown>
        </div>
      )}
      <section>{children && <div className={css.content}>{children}</div>}</section>
    </div>
  );
};

export default Card;
