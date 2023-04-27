import { Tooltip as AntdTooltip } from 'antd';
import React, { ReactNode } from 'react';

export interface TooltipProps {
  children?: ReactNode;
  mouseEnterDelay?: number;
  placement?:
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
  content?: ReactNode;
  trigger?: 'hover' | 'focus' | 'click' | 'contextMenu' | Array<string>;
  showArrow?: boolean;
}

const Tooltip: React.FC<TooltipProps> = ({
  mouseEnterDelay = 0.1,
  placement = 'top',
  content,
  ...props
}: TooltipProps) => {
  return (
    <AntdTooltip
      mouseEnterDelay={mouseEnterDelay}
      placement={placement}
      title={content}
      {...props}
    />
  );
};
export default Tooltip;
