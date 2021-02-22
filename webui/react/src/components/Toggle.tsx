import { Switch } from 'antd';
import React, { useCallback } from 'react';

import Label from './Label';
import css from './Toggle.module.scss';

interface Props {
  checked?: boolean;
  onChange?: (checked: boolean) => void;
  prefixLabel?: string;
  suffixLabel?: string;
}

const Toggle: React.FC<Props> = ({ checked = false, onChange, ...props }: Props) => {
  const handleClick = useCallback(() => {
    if (onChange) onChange(!checked);
  }, [ checked, onChange ]);

  return (
    <div className={css.base} onClick={handleClick}>
      {props.prefixLabel && <Label>{props.prefixLabel}</Label>}
      <Switch checked={checked} />
      {props.suffixLabel && <Label>{props.suffixLabel}</Label>}
    </div>
  );
};

export default Toggle;
