import { Space, Switch } from 'antd';
import React, { useCallback } from 'react';

import Label from 'components/kit/internal/Label';

interface Props {
  checked?: boolean;
  label?: string;
  onChange?: (checked: boolean) => void;
}

const Toggle: React.FC<Props> = ({ checked = false, label, onChange }: Props) => {
  const handleClick = useCallback(() => {
    if (onChange) onChange(!checked);
  }, [checked, onChange]);

  return (
    <Space onClick={handleClick}>
      {label && <Label>{label}</Label>}
      <Switch checked={checked} size="small" />
    </Space>
  );
};

export default Toggle;
