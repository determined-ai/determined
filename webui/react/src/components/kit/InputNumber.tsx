import { InputNumber as AntdInputNumber, Form } from 'antd';
import React from 'react';

import { FormItemWrapper, WrapperProps } from './Input';

type LabelCol = {
  span: number;
};

interface InputNumberProps {
  defaultValue?: number;
  disabled?: boolean;
  label: string;
  labelCol?: LabelCol; // https://ant.design/components/grid#col
  max?: number;
  min?: number;
  onChange?: () => void;
  precision?: number;
  step?: number;
  value?: number;
}

type WrappedInputNumberProps = WrapperProps & InputNumberProps;
const InputNumber: React.FC<WrappedInputNumberProps> = ({
  noForm,
  ...props
}: WrappedInputNumberProps) => {
  if (noForm) {
    return (
      <Form>
        <FormItemWrapper {...props}>
          <AntdInputNumber {...props} />
        </FormItemWrapper>
      </Form>
    );
  } else {
    return (
      <FormItemWrapper {...props}>
        <AntdInputNumber {...props} />
      </FormItemWrapper>
    );
  }
};
export default InputNumber;
