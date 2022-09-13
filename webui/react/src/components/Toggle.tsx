import { Space, Switch } from 'antd';
import React, { useCallback } from 'react';

import Label from './Label';

interface Props {
  checked?: boolean;
  onChange?: (checked: boolean) => void;
  prefixLabel?: string;
  suffixLabel?: string;
}

const Toggle: React.FC<Props> = ({ checked = false, onChange, ...props }: Props) => {
  const handleClick = useCallback(() => {
    if (onChange) onChange(!checked);
  }, [checked, onChange]);

  return (
    <Space onClick={handleClick}>
      {props.prefixLabel && <Label>{props.prefixLabel}</Label>}
      <Switch checked={checked} size="small" />
      {props.suffixLabel && <Label>{props.suffixLabel}</Label>}
    </Space>
  );
};

export default Toggle;
