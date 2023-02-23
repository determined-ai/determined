import { Dropdown, MenuProps } from 'antd';
import React from 'react';

import { ConditionalWrapper } from 'components/ConditionalWrapper';
import Link from 'components/Link';
import Icon from 'shared/components/Icon';

import Button from './Button';
import css from './Card.module.scss';

interface CardProps {
  actionMenu?: MenuProps;
  children?: React.ReactNode;
  disabled?: boolean;
  height?: number;
  onClick?: string | (() => void);
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
  const isLink = typeof onClick === 'string' && onClick.length > 0;
  if (onClick) classnames.push(css.clickable);
  const actionsAvailable = actionMenu?.items?.length !== undefined && actionMenu.items.length > 0;

  return (
    <ConditionalWrapper
      condition={isLink}
      wrapper={(children) =>
        isLink ? (
          <Link path={onClick} rawLink>
            {children}
          </Link>
        ) : (
          children // This branch should never be reached but the ternary satisfies Typescript
        )
      }>
      <div
        className={classnames.join(' ')}
        style={{ minHeight: `${height}px`, width: `${width}px` }}
        tabIndex={onClick ? 0 : -1}
        onClick={typeof onClick === 'function' ? onClick : undefined}>
        {children && <section className={css.content}>{children}</section>}
        {actionsAvailable && (
          <div className={css.action}>
            <Dropdown
              disabled={disabled}
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
    <div className={css.group} style={{ flexWrap: wrap ? 'wrap' : 'nowrap' }}>
      {children}
    </div>
  );
};

Card.Group = CardGroup;

export default Card;
