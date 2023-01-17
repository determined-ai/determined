import { InputNumber as AntdInputNumber, Form } from 'antd';
import React from 'react';

type LabelCol = {
  span: number;
};

interface NumberInputProps {
  defaultValue?: number;
  disabled?: boolean;
  label: string;
  labelCol?: LabelCol; // https://ant.design/components/grid#col
  max?: number;
  min?: number;
  onChange?: () => void;
  precision?: number;
  step?: number;
  value?: number;
}

const NumberInput: React.FC<NumberInputProps> = ({
  label,
  labelCol = { span: 24 },
  ...props
}: NumberInputProps) => {
  return (
    <Form.Item label={label} labelCol={labelCol}>
      <AntdInputNumber {...props} />
    </Form.Item>
  );
};
export default NumberInput;
