import { Button as AntdButton } from 'antd';
import React, { forwardRef, MouseEvent, ReactNode } from 'react';

import Tooltip from 'components/kit/Tooltip';
import { ConditionalWrapper } from 'components/kit/utils/components/ConditionalWrapper';

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
  ref?: React.Ref<HTMLElement>;
  shape?: 'circle' | 'default' | 'round';
  size?: 'large' | 'middle' | 'small';
  type?: 'primary' | 'link' | 'text' | 'ghost' | 'default' | 'dashed';
  tooltip?: string;
}

const Button: React.FC<ButtonProps> = forwardRef(
  (
    { shape = 'default', size = 'middle', type = 'default', tooltip = '', ...props }: ButtonProps,
    ref,
  ) => {
    return (
      <ConditionalWrapper
        condition={tooltip.length > 0}
        wrapper={(children) => <Tooltip content={tooltip}>{children}</Tooltip>}>
        <AntdButton
          className={css.base}
          ref={ref}
          shape={shape}
          size={size}
          tabIndex={props.disabled ? -1 : 0}
          type={type}
          {...props}
        />
      </ConditionalWrapper>
    );
  },
);

export default Button;
