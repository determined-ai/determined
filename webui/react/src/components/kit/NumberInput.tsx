import { InputNumber as AntdInputNumber, Form } from 'antd';
import React from 'react';

import { FormItemWrapper, WrapperProps } from './Input';

type LabelCol = {
  span: number;
};

interface NumberInputProps {
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

type WrappedNumberInputProps = WrapperProps & NumberInputProps;
const NumberInput: React.FC<WrappedNumberInputProps> = ({
  noForm,
  ...props
}: WrappedNumberInputProps) => {
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
export default NumberInput;
