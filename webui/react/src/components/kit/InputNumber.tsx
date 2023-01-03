import { InputNumber as AntdInputNumber } from 'antd';
import React from 'react';

interface InputNumberProps {
  className?: string;
  defaultValue?: number;
  disabled?: boolean;
  min?: number;
  onChange?: () => void;
  precision?: number;
  value?: number;
}

const InputNumber: React.FC<InputNumberProps> = (props: InputNumberProps) => {
  return <AntdInputNumber {...props} />;
};
export default InputNumber;
