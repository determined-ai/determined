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
  hint?: React.ReactNode;
  onClick?: () => void;
  title?: React.ReactNode;
}

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const Card: React.FC<CardProps> = ({
  actionMenu,
  children,
  footer,
  hint,
  onClick,
  title,
}: CardProps) => {
  return (
    <ConditionalWrapper
      condition={!!actionMenu}
      wrapper={(children) => (
        <Dropdown menu={actionMenu} placement="bottomLeft" trigger={['contextMenu']}>
          {children}
        </Dropdown>
      )}>
      <div className={css.base} tabIndex={onClick ? 0 : -1} onClick={onClick}>
        {(!!title || !!actionMenu || !!hint) && (
          <div className={css.header}>
            <Typography.Title className={css.title} ellipsis={{ rows: 1, tooltip: true }} level={5}>
              {title}
            </Typography.Title>
            <div className={css.extraArea}>
              {!!actionMenu && (
                <div className={css.action}>
                  <Dropdown menu={actionMenu} placement="bottomRight" trigger={['click']}>
                    <Button type="text" onClick={stopPropagation}>
                      <Icon name="overflow-horizontal" />
                    </Button>
                  </Dropdown>
                </div>
              )}
              {!!hint && <div className={css.hint}>{hint}</div>}
            </div>
          </div>
        )}
        <div className={css.content}>{children}</div>
        <div className={css.footer}>{footer}</div>
      </div>
    </ConditionalWrapper>
  );
};

export default Card;
