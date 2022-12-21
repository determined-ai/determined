import { Tooltip } from 'antd';
import React, { ReactNode } from 'react';

interface TooltipProps {
  children?: ReactNode;
  mouseEnterDelay?: number;
  placement?: 'top' | 'left' | 'right' | 'bottom' | 'topLeft' | 'topRight' | 'bottomLeft' | 'bottomRight' | 'leftTop' | 'leftBottom' | 'rightTop' | 'rightBottom';
  title?: string;
  trigger?: 'hover' | 'focus' | 'click' | 'contextMenu' | Array<string>;
}

const TooltipComponent: React.FC<TooltipProps> = ({ mouseEnterDelay = 0.1, placement = 'top', ...props }: TooltipProps) => {
  return (
    <Tooltip mouseEnterDelay={mouseEnterDelay} placement={placement} {...props} />
  );
};
export default TooltipComponent;
