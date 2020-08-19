import React, { PropsWithChildren, useCallback, useEffect, useMemo, useState } from 'react';

import { toRem } from 'utils/dom';
import { generateLetters } from 'utils/string';

import css from './DropdownMenu.module.scss';

export enum Placement {
  Bottom = 'bottom',
  BottomLeft = 'bottomLeft',
  BottomRight = 'bottomRight',
  Left = 'left',
  LeftBottom = 'leftBottom',
  LeftTop = 'leftTop',
  Right = 'right',
  RightTop = 'rightTop',
  RightBottom = 'rightBottom',
  Top = 'top',
  TopLeft = 'topLeft',
  TopRight = 'topRight',
}

interface Props {
  menu: React.ReactNode;
  offset?: { x: number, y: number };
  onVisibleChange?: (visible: boolean) => void;
  placement?: Placement;
  showArrow?: boolean;
}

const DropdownMenu: React.FC<Props> = ({
  offset = { x: 0, y: 0 },
  onVisibleChange,
  placement = Placement.BottomLeft,
  showArrow = true,
  ...props
}: PropsWithChildren<Props>) => {
  const [ id ] = useState(generateLetters());
  const [ isVisible, setIsVisible ] = useState(false);
  const classes = [ css.base, css[placement] ];

  const menuStyle = useMemo(() => {
    switch (placement) {
      case Placement.Bottom:
        return {
          left: '50%',
          top: `calc(100% + ${toRem(offset.y)})`,
          transform: 'translateX(-50%)',
        };
      case Placement.BottomLeft:
        return {
          left: toRem(offset.x),
          top: `calc(100% + ${toRem(offset.y)})`,
        };
      case Placement.BottomRight:
        return {
          right: toRem(offset.x),
          top: `calc(100% + ${toRem(offset.y)})`,
        };
      case Placement.Left:
        return {
          right: `calc(100% - ${toRem(offset.x)})`,
          top: '50%',
          transform: 'translateY(-50%)',
        };
      case Placement.LeftBottom:
        return {
          bottom: toRem(offset.y),
          right: `calc(100% - ${toRem(offset.x)})`,
        };
      case Placement.LeftTop:
        return {
          right: `calc(100% - ${toRem(offset.x)})`,
          top: toRem(offset.y),
        };
      case Placement.Right:
        return {
          left: `calc(100% + ${toRem(offset.x)})`,
          top: '50%',
          transform: 'translateY(-50%)',
        };
      case Placement.RightBottom:
        return {
          bottom: toRem(offset.y),
          left: `calc(100% + ${toRem(offset.x)})`,
        };
      case Placement.RightTop:
        return {
          left: `calc(100% + ${toRem(offset.x)})`,
          top: toRem(offset.y),
        };
      case Placement.Top:
        return {
          bottom: `calc(100% - ${toRem(offset.y)})`,
          left: '50%',
          transform: 'translateX(-50%)',
        };
      case Placement.TopLeft:
        return {
          bottom: `calc(100% - ${toRem(offset.y)})`,
          left: toRem(offset.x),
        };
      case Placement.TopRight:
        return {
          bottom: `calc(100% - ${toRem(offset.y)})`,
          right: toRem(offset.x),
        };
      default:
        return undefined;
    }
  }, [ offset, placement ]);

  if (isVisible) classes.push(css.visible);
  if (showArrow) classes.push(css.arrow);

  const handleClick = useCallback((event) => {
    if (!event || !event.target) return;

    event.stopPropagation();

    const isTrigger = event.target.closest(`#${id}`);

    if (isTrigger) {
      setIsVisible(prev => !prev);
    } else {
      setIsVisible(false);
    }
  }, [ id ]);

  useEffect(() => {
    if (onVisibleChange) onVisibleChange(isVisible);
  }, [ isVisible, onVisibleChange ]);

  useEffect(() => {
    document.addEventListener('click', handleClick);
    return () => document.removeEventListener('click', handleClick);
  }, [ handleClick ]);

  return <div className={classes.join(' ')}>
    <div className={css.menu} style={menuStyle}>{props.menu}</div>
    <div className={css.trigger} id={id}>{props.children}</div>
  </div>;
};

export default DropdownMenu;
