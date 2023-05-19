import { Button as AntdButton } from 'antd';
import React, { forwardRef, MouseEvent, ReactNode } from 'react';

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
  column?: boolean;
  loading?: boolean | { delay?: number };
  onClick?: (event: MouseEvent) => void;
  ref?: React.Ref<HTMLElement>;
  selected?: boolean;
  size?: 'large' | 'middle' | 'small';
  type?: 'primary' | 'text' | 'default' | 'dashed';
  tooltip?: string;
}

const Button: React.FC<ButtonProps> = forwardRef(
  ({ size = 'middle', tooltip = '', ...props }: ButtonProps, ref) => {
    const classes = [css.base];
    if (props.selected) classes.push(css.selected);
    if (props.column) classes.push(css.column);
    return (
      <ConditionalWrapper
        condition={tooltip.length > 0}
        wrapper={(children) => <Tooltip content={tooltip}>{children}</Tooltip>}>
        <AntdButton
          className={classes.join(' ')}
          ref={ref}
          size={size}
          tabIndex={props.disabled ? -1 : 0}
          {...props}
        />
      </ConditionalWrapper>
    );
  },
);

export default Button;
