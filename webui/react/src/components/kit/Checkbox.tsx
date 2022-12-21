import { Checkbox } from 'antd';
import React, { ReactNode } from 'react';

interface CheckboxProps {
  checked?: boolean;
  children?: ReactNode;
  disabled?: boolean;
  indeterminate?: boolean;
  onChange?: () => void;
}

const CheckboxComponent: React.FC<CheckboxProps> = (props: CheckboxProps) => {
  return (
    <Checkbox {...props} />
  );
};

export default CheckboxComponent;
