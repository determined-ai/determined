import { Input as AntdInput, InputProps, InputRef } from 'antd';
import React, { forwardRef, ForwardRefExoticComponent, RefAttributes } from 'react';

import InputNumber from './InputNumber';
import InputSearch from './InputSearch';

const Input: Input = forwardRef<InputRef, InputProps>((props: InputProps, ref) => {
  return (
    <AntdInput ref={ref} {...props} />
  );
}) as Input;

type Input = ForwardRefExoticComponent<InputProps & RefAttributes<InputRef>> & {
  Group: typeof AntdInput.Group,
  Number: typeof InputNumber,
  Password: typeof AntdInput.Password,
  Search: typeof InputSearch,
  TextArea: typeof AntdInput.TextArea,
};

Input.Group = AntdInput.Group;
Input.Number = InputNumber;
Input.Password = AntdInput.Password;
Input.Search = InputSearch;
Input.TextArea = AntdInput.TextArea;

export default Input;
