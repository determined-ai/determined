import { Form } from 'antd';
import React from 'react';

import SelectFilter, { Props as SelectProps } from 'components/SelectFilter';

import { FormItemWrapper, WrapperProps } from './Input';

type WrappedInputSelectProps = WrapperProps & SelectProps;
const InputSelect: React.FC<WrappedInputSelectProps> = ({
  label,
  noForm,
  ...props
}: WrappedInputSelectProps) => {
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
export default InputSelect;
