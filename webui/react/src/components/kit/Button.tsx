import { Button as AntdButton } from 'antd';
import React, { CSSProperties, MouseEvent, ReactNode } from 'react';

interface ButtonProps {
  children?: ReactNode;
  danger?: boolean;
  disabled?: boolean;
  ghost?: boolean;
  icon?: ReactNode;
  loading?: boolean | { delay?: number };
  onClick?: (event: MouseEvent) => void;
  shape?: 'circle' | 'default' | 'round';
  size?: 'large' | 'middle' | 'small';
  // TODO: remove style prop after adding iconic button support to component
  style?: CSSProperties;
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
