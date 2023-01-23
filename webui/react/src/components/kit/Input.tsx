import {
  Input as AntdInput,
  InputRef as AntdInputRef,
} from 'antd';
import React, { CSSProperties, FC, forwardRef, ForwardRefExoticComponent, ReactNode, RefAttributes } from 'react';

interface InputProps {
  addonAfter?: ReactNode;
  allowClear?: boolean | { clearIcon: ReactNode };
  autoComplete?: string;
  autoFocus?: boolean;
  bordered?: boolean;
  className?: string;
  defaultValue?: string;
  disabled?: boolean;
  max?: number;
  maxLength?: number;
  min?: number;
  onBlur?: () => void;
  onChange?: () => void;
  onPressEnter?: () => void;
  placeholder?: string;
  prefix?: ReactNode;
  size?: 'large' | 'middle' | 'small';
  style?: CSSProperties;
  title?: string;
  type?: string;
  value?: string;
  width?: string | number;
}

interface TextAreaProps {
  disabled?: boolean;
  onChange?: () => void;
  placeholder?: string;
  rows?: number;
  value?: string;
}

interface PasswordProps {
  disabled?: boolean;
  placeholder?: string;
  prefix?: ReactNode;
}

interface GroupProps {
  className?: string;

  compact?: boolean;
}

const Input: Input = forwardRef<InputRef, InputProps>(
  (props: InputProps, ref) => {
    return (
      <AntdInput {...props} ref={ref} />
    );
  },
) as Input;

type Input = ForwardRefExoticComponent<InputProps & RefAttributes<AntdInputRef>> & {
  Group: FC<GroupProps>;
  Password: ForwardRefExoticComponent<PasswordProps & RefAttributes<AntdInputRef>>;
  TextArea: ForwardRefExoticComponent<TextAreaProps & RefAttributes<AntdInputRef>>;
};

Input.Group = AntdInput.Group;

Input.Password = forwardRef((props: PasswordProps, ref) => {
  return (
    <AntdInput.Password {...props} ref={ref} />
  );
});

Input.TextArea = forwardRef((props: TextAreaProps, ref) => {
  return (
    <AntdInput.TextArea {...props} ref={ref} />
  );
});

export type InputRef = AntdInputRef;

export default Input;
