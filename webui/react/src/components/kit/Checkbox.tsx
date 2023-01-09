import { Checkbox as AntdCheckbox } from 'antd';
import type { CheckboxChangeEvent } from 'antd/lib/checkbox';
import React, { ReactNode } from 'react';

interface CheckboxProps {
  checked?: boolean;
  children?: ReactNode;
  disabled?: boolean;
  indeterminate?: boolean;
  onChange?: (event: CheckboxChangeEvent) => void;
}

const Checkbox: React.FC<CheckboxProps> = (props: CheckboxProps) => {
  return <AntdCheckbox {...props} />;
};

export default Checkbox;
