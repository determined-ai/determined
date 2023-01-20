import { Form } from 'antd';
import React from 'react';

import SelectFilter, { Props as SelectProps } from 'components/SelectFilter';

import { FormItemWrapper, WrapperProps } from './Input';

type WrappedInputSelectProps = WrapperProps & SelectProps;
const InputSelect: React.FC<WrappedInputSelectProps> = ({
  label, // only pass label to FormItemWrapper, not also to SelectFilter
  ...props
}: WrappedInputSelectProps) => {
  return (
    <FormItemWrapper label={label} {...props}>
      <SelectFilter {...props} />
    </FormItemWrapper>
  );
};
export default InputSelect;
