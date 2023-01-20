import { Button as AntdButton } from 'antd';
import React, { MouseEvent, ReactNode } from 'react';

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
}

const Button: React.FC<ButtonProps> = ({
  shape = 'default',
  size = 'middle',
  type = 'default',
  ...props
}: ButtonProps) => {
  return <AntdButton shape={shape} size={size} type={type} {...props} />;
};

export default Button;
