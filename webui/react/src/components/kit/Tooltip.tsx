import { Tooltip as AntdTooltip } from 'antd';
import React, { ReactNode } from 'react';

interface TooltipProps {
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
  title?: ReactNode;
  trigger?: 'hover' | 'focus' | 'click' | 'contextMenu' | Array<string>;
}

const Tooltip: React.FC<TooltipProps> = ({
  mouseEnterDelay = 0.1,
  placement = 'top',
  ...props
}: TooltipProps) => {
  return <AntdTooltip mouseEnterDelay={mouseEnterDelay} placement={placement} {...props} />;
};
export default Tooltip;
