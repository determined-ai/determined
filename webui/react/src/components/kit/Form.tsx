import {
  Form as AntdForm,
  FormInstance as AntdFormInstance,
  FormItemProps as AntdFormItemProps,
} from 'antd';
import { FormListFieldData as AntdFormListFieldData } from 'antd/lib/form/FormList';
import { FieldData as AntdFieldData, NamePath as AntdNamePath } from 'rc-field-form/lib/interface';
import React, { FC, ReactNode, Ref, RefObject, useRef } from 'react';

type Rules = AntdFormItemProps['rules']; // https://github.com/ant-design/ant-design/issues/39466
type GridCol = {
  span: number;
};
type TriggerEvent = 'onChange' | 'onSubmit';

interface FormItemProps {
  children?: ReactNode;
  dependencies?: AntdNamePath[];
  extra?: ReactNode;
  field?: AntdFormListFieldData;
  hidden?: boolean;
  initialValue?: string;
  label?: string;
  labelCol?: GridCol; // https://ant.design/components/grid#col
  max?: number;
  maxMessage?: string;
  name?: string;
  noStyle?: boolean;
  required?: boolean;
  requiredMessage?: string;
  rules?: Rules; // https://ant.design/components/form#rule
  validateMessage?: string;
  validateStatus?: 'success' | 'warning' | 'error' | 'validating';
  validateTrigger?: TriggerEvent[];
  valuePropName?: string;
}

const FormItem: React.FC<FormItemProps> = ({
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
}: FormItemProps) => {
  if (required) rules.push({ message: requiredMessage || `${label} required`, required: true });
  if (max) rules.push({ max, message: maxMessage || `${label} cannot exceed ${max} characters` });

  return (
    <AntdForm.Item
      help={validateMessage}
      label={label}
      labelCol={labelCol}
      name={name}
      required={required}
      rules={rules}
      validateStatus={validateStatus}
      validateTrigger={validateTrigger}>
      {children}
    </AntdForm.Item>
  );
};

interface FormProps {
  autoComplete?: string;
  children?: ReactNode;
  className?: string;
  fields?: AntdFieldData[];
  form?: AntdFormInstance;
  hidden?: boolean;
  id?: string;
  initialValues?: object;
  labelCol?: GridCol;
  layout?: 'horizontal' | 'vertical' | 'inline';
  name?: string
  onFieldsChange?: () => void;
  onFinish?: () => void;
  onValuesChange?: () => void;
  ref?: Ref<AntdFormInstance>
  wrapperCol?: GridCol;
}

type Form = JSX.Element & {
  Item?: FC<FormItemProps>;
  List?: typeof AntdForm.List;
  useForm?: typeof AntdForm.useForm;
}

const Form = (props: FormProps): JSX.Element => {
  return <AntdForm {...props} />;
};

Form.Item = FormItem;
Form.List = AntdForm.List;
Form.useForm = AntdForm.useForm;

export const useFormInstance = (): RefObject<AntdFormInstance> => {
  return useRef<AntdFormInstance>(null);
};

export default Form;
