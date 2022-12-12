import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { ValueOf } from 'shared/types';

import css from './Dropdown.module.scss';

export const Placement = {
  Bottom: 'bottom',
  BottomLeft: 'bottomLeft',
  BottomRight: 'bottomRight',
  Left: 'left',
  LeftBottom: 'leftBottom',
  LeftTop: 'leftTop',
  Right: 'right',
  RightBottom: 'rightBottom',
  RightTop: 'rightTop',
  Top: 'top',
  TopLeft: 'topLeft',
  TopRight: 'topRight',
} as const;

export type Placement = ValueOf<typeof Placement>;

interface Props {
  children: React.ReactNode;
  content: React.ReactNode;
  disableAutoDismiss?: boolean;
  initVisible?: boolean;
  offset?: { x: number; y: number };
  onVisibleChange?: (visible: boolean) => void;
  placement?: Placement;
  showArrow?: boolean;
}

const Dropdown: React.FC<Props> = ({
  disableAutoDismiss = false,
  offset = { x: 0, y: 0 },
  initVisible = false,
  onVisibleChange,
  placement = Placement.BottomLeft,
  showArrow = true,
  ...props
}: Props) => {
  const [isVisible, setIsVisible] = useState(initVisible);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLDivElement>(null);
  const classes = [css.base, css[placement]];

  const contentStyle = useMemo(() => {
    switch (placement) {
      case Placement.Bottom:
        return {
          left: '50%',
          top: `calc(100% + ${offset.y}px)`,
          transform: 'translateX(-50%)',
        };
      case Placement.BottomLeft:
        return {
          left: `${offset.x}px`,
          top: `calc(100% + ${offset.y}px)`,
        };
      case Placement.BottomRight:
        return {
          right: `${offset.x}px`,
          top: `calc(100% + ${offset.y}px)`,
        };
      case Placement.Left:
        return {
          right: `calc(100% - ${offset.x}px)`,
          top: '50%',
          transform: 'translateY(-50%)',
        };
      case Placement.LeftBottom:
        return {
          bottom: `${offset.y}px`,
          right: `calc(100% - ${offset.x}px)`,
        };
      case Placement.LeftTop:
        return {
          right: `calc(100% - ${offset.x}px)`,
          top: `${offset.y}px`,
        };
      case Placement.Right:
        return {
          left: `calc(100% + ${offset.x}px)`,
          top: '50%',
          transform: 'translateY(-50%)',
        };
      case Placement.RightBottom:
        return {
          bottom: `${offset.y}px`,
          left: `calc(100% + ${offset.x}px)`,
        };
      case Placement.RightTop:
        return {
          left: `calc(100% + ${offset.x}px)`,
          top: `${offset.y}px`,
        };
      case Placement.Top:
        return {
          bottom: `calc(100% - ${offset.y}px)`,
          left: '50%',
          transform: 'translateX(-50%)',
        };
      case Placement.TopLeft:
        return {
          bottom: `calc(100% - ${offset.y}px)`,
          left: `${offset.x}px`,
        };
      case Placement.TopRight:
        return {
          bottom: `calc(100% - ${offset.y}px)`,
          right: `${offset.x}px`,
        };
      default:
        return undefined;
    }
  }, [offset, placement]);

  if (isVisible) classes.push(css.visible);
  if (showArrow) classes.push(css.arrow);

  const handleClick = useCallback(
    (event: Event) => {
      if (!event || !event.target) return;

      event.stopPropagation();

      const target = event.target as Element
      const isAntPicker =
        target.closest('div') &&
        typeof target.closest('div')?.className === 'string' &&
        target.closest('div')?.className?.indexOf('ant-picker') || -1 >= 0;
      const isTrigger = triggerRef.current?.contains(target);
      const isDropdown = dropdownRef.current?.contains(target);

      if (isAntPicker) {
        return;
      } else if (isTrigger) {
        setIsVisible((prev) => !prev);
      } else if (isDropdown) {
        if (!disableAutoDismiss) setIsVisible(false);
      } else {
        setIsVisible(false);
      }
    },
    [disableAutoDismiss],
  );

  useEffect(() => {
    if (onVisibleChange) onVisibleChange(isVisible);
  }, [isVisible, onVisibleChange]);

  useEffect(() => {
    document.addEventListener('click', handleClick);
    return () => document.removeEventListener('click', handleClick);
  }, [handleClick]);

  return (
    <div className={classes.join(' ')}>
      <div className={css.content} ref={dropdownRef} style={contentStyle}>
        {props.content}
      </div>
      <div className={css.trigger} ref={triggerRef}>
        {props.children}
      </div>
    </div>
  );
};

export default Dropdown;
