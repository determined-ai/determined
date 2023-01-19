import { Form } from 'antd';
import React from 'react';

import SelectFilter, { Props as SelectProps } from 'components/SelectFilter';

import { FormItemWrapper, WrapperProps } from './Input';

type WrappedSelectInputProps = WrapperProps & SelectProps;
const SelectInput: React.FC<WrappedSelectInputProps> = ({
  label,
  noForm,
  ...props
}: WrappedSelectInputProps) => {
  if (noForm) {
    return (
      <Form>
        <FormItemWrapper label={label} {...props}>
          <SelectFilter {...props} />
        </FormItemWrapper>
      </Form>
    );
  } else {
    return (
      <FormItemWrapper label={label} {...props}>
        <SelectFilter {...props} />
      </FormItemWrapper>
    );
  }
};
export default SelectInput;
