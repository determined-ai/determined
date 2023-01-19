import {
  Input as AntdInput,
  InputProps as AntdInputProps,
  Form,
  FormItemProps,
  InputRef,
} from 'antd';
import { PasswordProps as AntdPasswordProps } from 'antd/lib/input/Password';
import { TextAreaProps as AntdTextAreaProps, TextAreaRef } from 'antd/lib/input/TextArea';
import React, { forwardRef, ForwardRefExoticComponent, ReactNode, RefAttributes } from 'react';

type Rules = FormItemProps['rules']; // https://github.com/ant-design/ant-design/issues/39466
type LabelCol = {
  span: number;
};
type TriggerEvent = 'onChange' | 'onSubmit';

export interface WrapperProps {
  children?: ReactNode;
  label?: string;
  labelCol?: LabelCol; // https://ant.design/components/grid#col
  max?: number;
  maxMessage?: string;
  name?: string;
  noForm?: boolean; // if not wrapped in an antd <Form> component
  required?: boolean;
  requiredMessage?: string;
  rules?: Rules; // https://ant.design/components/form#rule
  validateMessage?: string;
  validateStatus?: '' | 'success' | 'warning' | 'error' | 'validating' | undefined;
  validateTrigger?: TriggerEvent[];
}

export const FormItemWrapper: React.FC<WrapperProps> = ({
  children,
  label,
  labelCol = { span: 24 },
  name,
  rules = [],
  required,
  requiredMessage,
  max,
  maxMessage,
  validateMessage,
  validateTrigger,
  validateStatus,
}: WrapperProps) => {
  if (required) rules.push({ message: requiredMessage || `${label} required`, required: true });
  if (max) rules.push({ max, message: maxMessage || `${label} cannot exceed ${max} characters` });

  return (
    <Form.Item
      help={validateMessage}
      label={label}
      labelCol={labelCol}
      name={name}
      required={required}
      rules={rules}
      validateStatus={validateStatus}
      validateTrigger={validateTrigger}>
      {children}
    </Form.Item>
  );
};

type WrappedInputProps = AntdInputProps & WrapperProps;
const Input: Input = forwardRef<InputRef, WrappedInputProps>(
  ({ noForm, ...props }: WrappedInputProps, ref) => {
    if (noForm) {
      return (
        <Form>
          <FormItemWrapper max={255} {...props}>
            <AntdInput {...props} ref={ref} />
          </FormItemWrapper>
          ;
        </Form>
      );
    } else {
      return (
        <FormItemWrapper max={255} {...props}>
          <AntdInput {...props} ref={ref} />
        </FormItemWrapper>
      );
    }
  },
) as Input;

type Input = ForwardRefExoticComponent<WrappedInputProps & RefAttributes<InputRef>> & {
  Group: typeof AntdInput.Group;
  Password: ForwardRefExoticComponent<WrappedPasswordProps & RefAttributes<InputRef>>;
  TextArea: ForwardRefExoticComponent<WrappedTextAreaProps & RefAttributes<TextAreaRef>>;
};

Input.Group = AntdInput.Group;

type WrappedPasswordProps = AntdPasswordProps & WrapperProps;
Input.Password = React.forwardRef(({ noForm, ...props }: WrappedPasswordProps, ref) => {
  if (noForm) {
    return (
      <Form>
        <FormItemWrapper max={255} {...props}>
          <AntdInput.Password {...props} ref={ref} />
        </FormItemWrapper>
      </Form>
    );
  } else {
    return (
      <FormItemWrapper max={255} {...props}>
        <AntdInput.Password {...props} ref={ref} />
      </FormItemWrapper>
    );
  }
});

type WrappedTextAreaProps = AntdTextAreaProps & WrapperProps;
Input.TextArea = React.forwardRef(({ noForm, ...props }: WrappedTextAreaProps, ref) => {
  if (noForm) {
    return (
      <Form>
        <FormItemWrapper max={255} {...props}>
          <AntdInput.TextArea {...props} ref={ref} />
        </FormItemWrapper>
      </Form>
    );
  } else {
    return (
      <FormItemWrapper max={255} {...props}>
        <AntdInput.TextArea {...props} ref={ref} />
      </FormItemWrapper>
    );
  }
});

export default Input;
