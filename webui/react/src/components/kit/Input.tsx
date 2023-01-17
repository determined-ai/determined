import { Input as AntdInput, InputProps as AntdInputProps, Form, InputRef } from 'antd';
import { FormItemProps } from 'antd';
import React, { forwardRef, ForwardRefExoticComponent, RefAttributes } from 'react';

type Rules = FormItemProps['rules']; // https://github.com/ant-design/ant-design/issues/39466
type LabelCol = {
  span: number;
};
type TriggerEvent = 'onChange' | 'onSubmit';

interface InputProps extends AntdInputProps {
  label: string;
  labelCol?: LabelCol; // https://ant.design/components/grid#col
  max?: number;
  maxMessage?: string;
  name: string;
  noForm?: boolean; // if not wrapped in an antd <Form> component
  required?: boolean;
  requiredMessage?: string;
  rules?: Rules; // https://ant.design/components/form#rule
  validateTrigger?: TriggerEvent[];
}

const FormItemWrapper = (({
  label,
  labelCol = { span: 24 },
  name,
  rules,
  ref,
  required,
  requiredMessage,
  max = 255,
  maxMessage,
  validateTrigger,
  ...props
}) => {
  const maxRule = { max, message: maxMessage || `${label} cannot exceed ${max} characters` };
  const itemRules = rules ? [...rules, maxRule] : [maxRule];
  if (required) itemRules.push({ message: requiredMessage || `${label} required`, required: true });
  return (
    <Form.Item
      label={label}
      labelCol={labelCol}
      name={name}
      required={required}
      rules={itemRules}
      validateTrigger={validateTrigger}>
      <AntdInput ref={ref} {...props} />
    </Form.Item>
  );
}) as Input;

const Input: Input = forwardRef<InputRef, InputProps>(({ noForm, ...props }: InputProps, ref) => {
  if (noForm) {
    return (
      <Form>
        <FormItemWrapper {...props} ref={ref} />;
      </Form>
    );
  } else {
    return <FormItemWrapper {...props} ref={ref} />;
  }
}) as Input;

type Input = ForwardRefExoticComponent<InputProps & RefAttributes<InputRef>> & {
  Group: typeof AntdInput.Group;
  Password: typeof AntdInput.Password;
};

Input.Group = AntdInput.Group;
Input.Password = AntdInput.Password;

export default Input;
