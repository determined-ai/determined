import { InputNumber as AntdInputNumber } from 'antd';
import React from 'react';

interface InputNumberProps {
  className?: string;
  defaultValue?: number;
  disabled?: boolean;
  max?: number;
  min?: number;
  onChange?: (value: number | string | null) => void;
  placeholder?: string;
  precision?: number;
  step?: number;
  value?: number;
  onPressEnter?: (e: React.KeyboardEvent) => void;
}

const InputNumber: React.FC<InputNumberProps> = (props: InputNumberProps) => {
  return <AntdInputNumber {...props} />;
};
export default InputNumber;
