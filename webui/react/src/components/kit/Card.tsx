import { Dropdown, MenuProps } from 'antd';
import React, { Children, CSSProperties } from 'react';

import { ConditionalWrapper } from 'components/ConditionalWrapper';
import Grid, { GridMode } from 'components/Grid';
import Icon from 'components/kit/Icon';
import Link from 'components/Link';
import { isNumber } from 'shared/utils/data';

import Button from './Button';
import css from './Card.module.scss';

type CardPropsBase = {
  actionMenu?: MenuProps;
  children?: React.ReactNode;
  disabled?: boolean;
  size?: keyof typeof CardSize;
};

const CardSize: Record<string, CSSProperties> = {
  medium: { minHeight: '110px', minWidth: '302px' },
  small: { minHeight: '64px', minWidth: '143px' },
} as const;

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
  const sizeStyle = CardSize[size];
  switch (size) {
    case 'small':
      classnames.push(css.smallCard);
      break;
    case 'medium':
      classnames.push(css.mediumCard);
      break;
  }

  const actionsAvailable = actionMenu?.items?.length !== undefined && actionMenu.items.length > 0;

  return (
    <ConditionalWrapper
      condition={!!href}
      // This falseWrapper is so styles work consistently whether or not the card has a link.
      falseWrapper={(children) => (
        <div
          className={classnames.join(' ')}
          style={sizeStyle}
          tabIndex={onClick ? 0 : -1}
          onClick={onClick}>
          {children}
        </div>
      )}
      wrapper={(children) => (
        <Link className={classnames.join(' ')} path={href} style={sizeStyle}>
          {children}
        </Link>
      )}>
      <>
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
      </>
    </ConditionalWrapper>
  );
};

interface CardGroupProps {
  children?: React.ReactNode;
  size?: keyof typeof CardSize; // This should match the size of cards in group.
  wrap?: boolean;
}

const CardGroup: React.FC<CardGroupProps> = ({
  children,
  wrap = true,
  size = 'small',
}: CardGroupProps) => {
  const cardSize = CardSize[size].minWidth;
  const minCardWidth = cardSize ? (isNumber(cardSize) ? cardSize : parseInt(cardSize)) : undefined;

  return (
    <Grid
      className={css.groupBase}
      count={Children.toArray(children).length}
      gap={16}
      minItemWidth={minCardWidth}
      mode={wrap ? GridMode.AutoFill : GridMode.ScrollableRow}>
      {children}
    </Grid>
  );
};

Card.Group = CardGroup;

export default Card;
