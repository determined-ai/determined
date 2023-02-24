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
  size?: 'small' | 'medium';
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
  href,
  onClick,
  size = 'small',
}: CardProps) => {
  const classnames = [css.cardBase];
  if (href || onClick) classnames.push(css.clickable);
  switch (size) {
    case 'small':
      classnames.push(css.small);
      break;
    case 'medium':
      classnames.push(css.medium);
      break;
  }

  const actionsAvailable = actionMenu?.items?.length !== undefined && actionMenu.items.length > 0;

  return (
    <ConditionalWrapper
      condition={!!href}
      // This falseWrapper is so styles work consistently whether or not the card has a link.
      falseWrapper={(children) => <div>{children}</div>}
      wrapper={(children) => (
        <Link path={href} rawLink>
          {children}
        </Link>
      )}>
      <div className={classnames.join(' ')} tabIndex={onClick ? 0 : -1} onClick={onClick}>
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
  const classnames = [css.groupBase];
  classnames.push(wrap ? css.wrap : css.noWrap);

  return <div className={classnames.join(' ')}>{children}</div>;
};

Card.Group = CardGroup;

export default Card;
