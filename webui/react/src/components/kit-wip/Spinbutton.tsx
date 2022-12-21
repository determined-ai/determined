import { InputNumber } from 'antd';
import React from 'react';

interface SpinbuttonProps {
  className?: string;
  defaultValue?: number;
  disabled?: boolean;
  min?: number;
  onChange?: () => void;
  precision?: number;
  value?: number;
}

const SpinbuttonComponent: React.FC<SpinbuttonProps> = (props: SpinbuttonProps) => {
  return (
    <InputNumber {...props} />
  );
};
export default SpinbuttonComponent;
