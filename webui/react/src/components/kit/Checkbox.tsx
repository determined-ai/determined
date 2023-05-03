import { Checkbox as AntdCheckbox } from 'antd';
import type { CheckboxChangeEvent } from 'antd/lib/checkbox';
import React, { ReactNode } from 'react';

interface CheckboxProps {
  checked?: boolean;
  children?: ReactNode;
  disabled?: boolean;
  id?: string;
  indeterminate?: boolean;
  onChange?: (event: CheckboxChangeEvent) => void;
}

interface GroupProps {
  children?: ReactNode;
}

const Checkbox: Checkbox = (props: CheckboxProps) => {
  return <AntdCheckbox {...props} />;
};

type Checkbox = React.FC<CheckboxProps> & {
  Group: React.FC<GroupProps>;
};

Checkbox.Group = AntdCheckbox.Group;

export default Checkbox;
