import { Input as AntdInput, InputProps, InputRef } from 'antd';
import React, { forwardRef, ForwardRefExoticComponent, RefAttributes } from 'react';

const Input: Input = forwardRef<InputRef, InputProps>((props: InputProps, ref) => {
  return <AntdInput ref={ref} {...props} />;
}) as Input;

type Input = ForwardRefExoticComponent<InputProps & RefAttributes<InputRef>> & {
  Group: typeof AntdInput.Group;
  Password: typeof AntdInput.Password;
};

Input.Group = AntdInput.Group;
Input.Password = AntdInput.Password;

export default Input;
