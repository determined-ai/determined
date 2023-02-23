import { Dropdown, MenuProps } from 'antd';
import React from 'react';

import { ConditionalWrapper } from 'components/ConditionalWrapper';
import Link from 'components/Link';
import Icon from 'shared/components/Icon';

import Button from './Button';
import css from './Card.module.scss';

type CardPropsBase = {
  actionMenu?: MenuProps;
  children?: React.ReactNode;
  disabled?: boolean;
  height?: number;
  width?: number;
};

type CardProps = (
  | {
      href?: string;
      onClick?: never;
    }
  | {
      href?: never;
      onClick?: () => void;
    }
) &
  CardPropsBase;

type Card = React.FC<CardProps> & {
  Group: React.FC<CardGroupProps>;
};

const stopPropagation = (e: React.MouseEvent): void => e.stopPropagation();

const Card: Card = ({
  actionMenu,
  children,
  disabled = false,
  onClick,
  href,
  height = 184,
  width = 184,
}: CardProps) => {
  const classnames = [css.base];
  if (href || onClick) classnames.push(css.clickable);
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
        tabIndex={onClick ? 0 : -1}
        onClick={onClick}>
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
