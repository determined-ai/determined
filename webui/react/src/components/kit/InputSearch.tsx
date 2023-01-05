import { Input } from 'antd';
import React from 'react';

interface InputSearchProps {
  allowClear?: boolean;
  disabled?: boolean;
  enterButton?: boolean;
  placeholder?: string;
  value?: string;
}

const InputSearch: React.FC<InputSearchProps> = (props: InputSearchProps) => {
  return <Input.Search {...props} />;
};

export default InputSearch;
