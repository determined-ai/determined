import { Button as AntdButton } from 'antd';
import React, { MouseEvent, ReactNode } from 'react';

import { ConditionalWrapper } from 'components/ConditionalWrapper';
import Tooltip from 'components/kit/Tooltip';

import css from './Button.module.scss';

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
  tooltip = '',
  ...props
}: ButtonProps) => {
  return (
    <ConditionalWrapper
      condition={tooltip.length > 0}
      wrapper={(children) => <Tooltip title={tooltip}>{children}</Tooltip>}>
      <AntdButton
        className={css.base}
        shape={shape}
        size={size}
        tabIndex={props.disabled ? -1 : 0}
        type={type}
        {...props}
      />
    </ConditionalWrapper>
  );
};

export default Button;
