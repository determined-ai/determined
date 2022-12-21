import { Input } from 'antd';
import React, { CSSProperties } from 'react';

interface SearchboxProps {
  allowClear?: boolean;
  disabled?: boolean;
  enterButton?: boolean;
  placeholder?: string;
  style: CSSProperties;
  value?: string;
}

const Searchbox: React.FC<SearchboxProps> = (props: SearchboxProps) => {
  return (
    <Input.Search {...props} />
  );
};
export default Searchbox;
