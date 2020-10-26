import { Switch } from 'antd';
import React, { useCallback, useState } from 'react';

import Label from './Label';
import css from './Toggle.module.scss';

interface Props {
  prefixLabel?: string;
  checked?: boolean;
  suffixLabel?: string;
  onChange?: (checked: boolean) => void;
}

const Toggle: React.FC<Props> = ({ onChange, ...props }: Props) => {
  const [ checked, setChecked ] = useState(props.checked || false);

  const handleClick = useCallback(() => {
    setChecked(!checked);
    if (onChange) onChange(!checked);
  }, [ checked, onChange ]);

  return (
    <div className={css.base} onClick={handleClick}>
      {props.prefixLabel && <Label style={{ textAlign: 'right' }}>{props.prefixLabel}</Label>}
      <Switch checked={checked} />
      {props.suffixLabel && <Label>{props.suffixLabel}</Label>}
    </div>
  );
};

export default Toggle;
