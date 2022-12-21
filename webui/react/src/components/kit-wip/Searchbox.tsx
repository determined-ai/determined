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

const SearchboxComponent: React.FC<SearchboxProps> = (props: SearchboxProps) => {
  return (
    <Input.Search {...props} />
  );
};
export default SearchboxComponent;
