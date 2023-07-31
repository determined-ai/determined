import { Popover as AntdPopover, Tooltip as AntdTooltip } from 'antd';
import React, { ReactNode } from 'react';

import { isString } from 'utils/data';

import css from './Tooltip.module.scss';

export type Placement =
  | 'top'
  | 'left'
  | 'right'
  | 'bottom'
  | 'topLeft'
  | 'topRight'
  | 'bottomLeft'
  | 'bottomRight'
  | 'leftTop'
  | 'leftBottom'
  | 'rightTop'
  | 'rightBottom';

export interface TooltipProps {
  children?: ReactNode;
  content?: ReactNode;
  mouseEnterDelay?: number;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  placement?: Placement;
  trigger?: 'hover' | 'focus' | 'click' | 'contextMenu' | Array<string>;
  showArrow?: boolean;
}

const Tooltip: React.FC<TooltipProps> = ({
  content,
  mouseEnterDelay,
  open,
  placement = 'top',
  ...props
}: TooltipProps) => {
  if (isString(content)) {
    return (
      <AntdTooltip // use default antd Tooltip styling for string content
        autoAdjustOverflow
        mouseEnterDelay={mouseEnterDelay}
        open={open}
        overlayClassName={css.content}
        placement={placement}
        title={content}
        {...props}
      />
    );
  }
  return (
    <AntdPopover // use default antd Popover styling for component content
      autoAdjustOverflow
      mouseEnterDelay={mouseEnterDelay}
      open={open}
      overlayClassName={css.content}
      placement={placement}
      title={content}
      {...props}
    />
  );
};
export default Tooltip;
