import { Dropdown, MenuProps, Space } from 'antd';
import React from 'react';

import { ConditionalWrapper } from 'components/ConditionalWrapper';
import Link from 'components/Link';
import Icon from 'shared/components/Icon';

import Button from './Button';
import css from './Card.module.scss';

interface CardProps {
  actionMenu?: MenuProps;
  children?: React.ReactNode;
  clickable?: boolean;
  disabled?: boolean;
  height?: number;
  href?: string;
  onClick?: () => void;
  width?: number;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const Card: Card = ({
  actionMenu,
  children,
  clickable,
  disabled = false,
  onClick,
  height = 184,
  width = 184,
  href,
}: CardProps) => {
  const classnames = [css.base];
  const clicky = onClick || clickable || href;
  if (clicky) classnames.push(css.clickable);
  const actionsAvailable = actionMenu?.items?.length !== undefined && actionMenu.items.length > 0;

  return (
    <ConditionalWrapper
      condition={!!href}
      wrapper={(children) => (
        <Link path={href} rawLink>
          {children}
        </Link>
      )}>
      <div
        className={classnames.join(' ')}
        style={{ minHeight: `${height}px`, width: `${width}px` }}
        tabIndex={clicky ? 0 : -1}
        onClick={onClick}>
        {children && <section className={css.content}>{children}</section>}
        {actionMenu && (
          <div className={css.action}>
            <Dropdown
              disabled={disabled || !actionsAvailable}
              menu={actionMenu}
              placement="bottomRight"
              trigger={['click']}>
              <Button size="small" type="text" onClick={stopPropagation}>
                <Icon name="overflow-horizontal" />
              </Button>
            </Dropdown>
          </div>
        )}
      </div>
    </ConditionalWrapper>
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
