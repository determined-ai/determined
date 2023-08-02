import { Input as AntdInput, InputRef as AntdInputRef } from 'antd';
import React, {
  CSSProperties,
  FC,
  forwardRef,
  ForwardRefExoticComponent,
  ReactNode,
  RefAttributes,
} from 'react';

interface InputProps {
  addonAfter?: ReactNode;
  allowClear?: boolean | { clearIcon: ReactNode };
  autoComplete?: string;
  autoFocus?: boolean;
  bordered?: boolean;
  className?: string;
  defaultValue?: string;
  disabled?: boolean;
  id?: string;
  max?: number;
  maxLength?: number;
  min?: number;
  onBlur?: (
    e: React.FocusEvent<HTMLInputElement> | React.KeyboardEvent<HTMLInputElement>,
    previousValue?: string,
  ) => void;
  onChange?: (e: React.ChangeEvent<HTMLInputElement>) => void;
  onPressEnter?: (e: React.KeyboardEvent<HTMLInputElement>) => void;
  placeholder?: string;
  prefix?: ReactNode;
  size?: 'large' | 'middle' | 'small';
  style?: CSSProperties;
  title?: string;
  type?: string;
  value?: string;
}

interface TextAreaProps {
  disabled?: boolean;
  onChange?: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
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
  children?: ReactNode;
  className?: string;
  compact?: boolean;
}

const Input: Input = forwardRef<AntdInputRef, InputProps>((props: InputProps, ref) => {
  return <AntdInput {...props} ref={ref} />;
}) as Input;

type Input = ForwardRefExoticComponent<InputProps & RefAttributes<AntdInputRef>> & {
  Group: FC<GroupProps>;
  Password: ForwardRefExoticComponent<PasswordProps & RefAttributes<AntdInputRef>>;
  TextArea: ForwardRefExoticComponent<TextAreaProps & RefAttributes<AntdInputRef>>;
};

Input.Group = AntdInput.Group;

Input.Password = forwardRef<AntdInputRef, PasswordProps>((props: PasswordProps, ref) => {
  return <AntdInput.Password {...props} ref={ref} />;
});

Input.TextArea = forwardRef<AntdInputRef, TextAreaProps>((props: TextAreaProps, ref) => {
  return <AntdInput.TextArea {...props} ref={ref} />;
});

export type InputRef = AntdInputRef;

export default Input;
