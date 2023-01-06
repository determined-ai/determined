import { Input } from 'antd';
import React from 'react';

interface SearchInputProps {
  allowClear?: boolean;
  disabled?: boolean;
  enterButton?: boolean;
  placeholder?: string;
  value?: string;
}

const SearchInput: React.FC<SearchInputProps> = (props: SearchInputProps) => {
  return <Input.Search {...props} />;
};

export default SearchInput;
