import { Input } from 'antd';
import React, { CSSProperties } from 'react';

interface InputSearchProps {
  allowClear?: boolean;
  disabled?: boolean;
  enterButton?: boolean;
  placeholder?: string;
  style: CSSProperties;
  value?: string;
}

const InputSearch: React.FC<InputSearchProps> = (props: InputSearchProps) => {
  return (
    <Input.Search {...props} />
  );
};

export default InputSearch;
