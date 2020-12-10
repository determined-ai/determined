import { Switch } from 'antd';
import React, { useCallback, useState } from 'react';

import Label from './Label';
import css from './Toggle.module.scss';

interface Props {
  checked?: boolean;
  onChange?: (checked: boolean) => void;
  prefixLabel?: string;
  suffixLabel?: string;
}

const Toggle: React.FC<Props> = ({ onChange, ...props }: Props) => {
  const [ checked, setChecked ] = useState(props.checked || false);

  const handleClick = useCallback(() => {
    setChecked(!checked);
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
