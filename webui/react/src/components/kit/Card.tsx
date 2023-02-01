import { Dropdown, MenuProps, Typography } from 'antd';
import React from 'react';

import { ConditionalWrapper } from 'components/ConditionalWrapper';
import Icon from 'shared/components/Icon';

import Button from './Button';
import css from './Card.module.scss';

interface CardProps {
  actionMenu?: MenuProps;
  children?: React.ReactNode;
  footer?: React.ReactNode;
  onClick?: () => void;
  title?: React.ReactNode;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const Card: React.FC<CardProps> = ({ actionMenu, children, footer, onClick, title }: CardProps) => {
  return (
    <ConditionalWrapper
      condition={!!actionMenu}
      wrapper={(children) => (
        <Dropdown menu={actionMenu} placement="bottomLeft" trigger={['contextMenu']}>
          {children}
        </Dropdown>
      )}>
      <div className={css.base} tabIndex={onClick ? 0 : -1} onClick={onClick}>
        {actionMenu && (
          <div className={css.action}>
            <Dropdown menu={actionMenu} placement="bottomRight" trigger={['click']}>
              <Button type="text" onClick={stopPropagation}>
                <Icon name="overflow-horizontal" />
              </Button>
            </Dropdown>
          </div>
        )}
        <section>
          {title && (
            <Typography.Title className={css.title} ellipsis={{ rows: 1, tooltip: true }} level={5}>
              {title}
            </Typography.Title>
          )}
          {children && <div className={css.content}>{children}</div>}
          {footer && <div className={css.footer}>{footer}</div>}
        </section>
      </div>
    </ConditionalWrapper>
  );
};

export default Card;
