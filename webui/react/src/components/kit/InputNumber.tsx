import { InputNumber as AntdInputNumber } from 'antd';
import React from 'react';

interface InputNumberProps {
  defaultValue?: number;
  disabled?: boolean;
  max?: number;
  min?: number;
  onChange?: () => void;
  precision?: number;
  step?: number;
  value?: number;
}

const InputNumber: React.FC<InputNumberProps> = (props: InputNumberProps) => {
  return (
    <AntdInputNumber {...props} />
  );
};
export default InputNumber;
