import { Button as AntdButton } from 'antd';
import React, { MouseEvent, ReactNode } from 'react';

import css from './Button.module.scss';
import Tooltip from './Tooltip';

interface ButtonProps {
  block?: boolean;
  children?: ReactNode;
  danger?: boolean;
  disabled?: boolean;
  ghost?: boolean;
  htmlType?: 'button' | 'submit' | 'reset';
  icon?: ReactNode;
  loading?: boolean | { delay?: number };
  onClick?: (event: MouseEvent) => void;
  shape?: 'circle' | 'default' | 'round';
  size?: 'large' | 'middle' | 'small';
  type?: 'primary' | 'link' | 'text' | 'ghost' | 'default' | 'dashed';
  tooltip?: string;
}

const Button: React.FC<ButtonProps> = ({
  shape = 'default',
  size = 'middle',
  type = 'default',
  ...props
}: ButtonProps) => {
  if (props.tooltip) {
    return (
      <Tooltip title={props.tooltip}>
        <AntdButton
          className={props.tooltip && css.wrapped}
          shape={shape}
          size={size}
          tabIndex={props.disabled ? -1 : 0}
          type={type}
          {...props}
        />
      </Tooltip>
    );
  }
  return (
    <AntdButton
      shape={shape}
      size={size}
      tabIndex={props.disabled ? -1 : 0}
      type={type}
      {...props}
    />
  );
};

export default Button;
