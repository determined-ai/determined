import { InputNumber as AntdInputNumber } from 'antd';
import React from 'react';

interface NumberInputProps {
  defaultValue?: number;
  disabled?: boolean;
  max?: number;
  min?: number;
  onChange?: () => void;
  precision?: number;
  step?: number;
  value?: number;
}

const NumberInput: React.FC<NumberInputProps> = (props: NumberInputProps) => {
  return <AntdInputNumber {...props} />;
};
export default NumberInput;
