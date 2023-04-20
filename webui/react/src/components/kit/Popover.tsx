import { Popover as AntdPopover } from 'antd';
import React, { ReactNode } from 'react';

export interface Props {
  content: ReactNode;
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
  trigger?: 'hover' | 'focus' | 'click' | 'contextMenu' | Array<string>;
  children?: ReactNode;
}

const Popover: React.FC<Props> = ({ placement = 'bottom', ...props }: Props) => {
  return <AntdPopover placement={placement} {...props} />;
};

export default Popover;
