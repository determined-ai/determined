import { InputNumber as AntdInputNumber } from 'antd';
import React, { forwardRef } from 'react';

interface InputNumberProps {
  className?: string;
  defaultValue?: number;
  disabled?: boolean;
  max?: number;
  min?: number;
  onChange?: (value: number | string | null) => void;
  onKeyDown?: (e: React.KeyboardEvent) => void;
  onPressEnter?: React.KeyboardEventHandler<HTMLInputElement>;
  placeholder?: string;
  precision?: number;
  step?: number;
  value?: number;
  autoFocus?: boolean;
  ref?: React.Ref<HTMLInputElement>;
}

const InputNumber: React.FC<InputNumberProps> = forwardRef(
  (props: InputNumberProps, ref?: React.Ref<HTMLInputElement>) => {
    return <AntdInputNumber {...props} ref={ref} />;
  },
);
export default InputNumber;
