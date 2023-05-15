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
  form?: string;
  htmlType?: 'button' | 'submit' | 'reset';
  icon?: ReactNode;
  loading?: boolean | { delay?: number };
  onClick?: (event: MouseEvent) => void;
  size?: 'large' | 'middle' | 'small';
  type?: 'primary' | 'link' | 'text' | 'default' | 'dashed';
  tooltip?: string;
}

const Button: React.FC<ButtonProps> = ({
  size = 'middle',
  tooltip = '',
  ...props
}: ButtonProps) => {
  return (
    <ConditionalWrapper
      condition={tooltip.length > 0}
      wrapper={(children) => <Tooltip content={tooltip}>{children}</Tooltip>}>
      <AntdButton
        className={css.base}
        size={size}
        tabIndex={props.disabled ? -1 : 0}
        {...props}
      />
    </ConditionalWrapper>
  );
};

export default Button;
