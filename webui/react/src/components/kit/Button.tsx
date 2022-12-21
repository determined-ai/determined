import { Button } from 'antd';
import React, { CSSProperties, MouseEvent, ReactNode } from 'react';

interface ButtonProps {
  children?: ReactNode;
  className?: string;
  danger?: boolean;
  disabled?: boolean;
  ghost?: boolean;
  icon?: ReactNode;
  loading?: boolean | { delay?: number };
  onClick?: (event: MouseEvent) => void;
  shape?: 'circle' | 'default' | 'round';
  size?: 'large' | 'middle' | 'small';
  style?: CSSProperties;
  type?: 'primary' | 'link' | 'text' | 'ghost' | 'default' | 'dashed';
}

const ButtonComponent: React.FC<ButtonProps> = ({ shape = 'default', size = 'middle', type = 'default', ...props }: ButtonProps) => {
  return (
    <Button shape={shape} size={size} type={type} {...props} />
  );
};

export default ButtonComponent;
