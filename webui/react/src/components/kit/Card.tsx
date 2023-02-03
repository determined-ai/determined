import { Dropdown, MenuProps, Space } from 'antd';
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

const Card: Card = ({
  actionMenu,
  children,
  disabled = false,
  onClick,
  height = 184,
  width = 184,
}: CardProps) => {
  const classnames = [css.base];
  if (onClick) classnames.push(css.clickable);
  const actionsAvailable = actionMenu?.items?.length !== undefined && actionMenu.items.length > 0;

  return (
    <div
      className={classnames.join(' ')}
      style={{ minHeight: `${height}px`, width: `${width}px` }}
      tabIndex={onClick ? 0 : -1}
      onClick={onClick}>
      {actionMenu && (
        <div className={css.action}>
          <Dropdown
            disabled={disabled || !actionsAvailable}
            menu={actionMenu}
            placement="bottomRight"
            trigger={['click']}>
            <Button type="text" onClick={stopPropagation}>
              <Icon name="overflow-horizontal" />
            </Button>
          </Dropdown>
        </div>
      )}
      {children && <section className={css.content}>{children}</section>}
    </div>
  );
};

type Card = React.FC<CardProps> & {
  Group: React.FC<CardGroupProps>;
};

interface CardGroupProps {
  children?: React.ReactNode;
  wrap?: boolean;
}

const CardGroup: React.FC<CardGroupProps> = ({ children, wrap = true }: CardGroupProps) => {
  return (
    <Space className={css.group} size="middle" wrap={wrap}>
      {children}
    </Space>
  );
};

Card.Group = CardGroup;

export default Card;
