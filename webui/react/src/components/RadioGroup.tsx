import { Radio } from 'antd';
import { RadioChangeEvent } from 'antd/lib/radio';
import React, { useCallback, useState } from 'react';

import Icon, { IconSize } from 'components/Icon';

import css from './RadioGroup.module.scss';

interface Props {
  className?: string;
  defaultOptionId?: string;
  onChange?: (id: string) => void;
  options: RadioGroupOption[];
}

export interface RadioGroupOption {
  icon?: string;
  iconSize?: IconSize;
  id: string;
  label?: string;
}

const RadioGroup: React.FC<Props> = ({ className, defaultOptionId, onChange, options }: Props) => {
  const [ selected, setSelected ] = useState<string | undefined>(() => defaultOptionId);
  const classes = [ css.base ];

  if (className) classes.push(className);

  const handleChange = useCallback((e: RadioChangeEvent) => {
    const id = e.target.value;
    setSelected(id);
    if (onChange) onChange(id);
  }, [ onChange ]);

  return (
    <Radio.Group className={classes.join(' ')} value={selected} onChange={handleChange}>
      {options.map(option => (
        <Radio.Button className={css.option} key={option.id} value={option.id}>
          {option.icon && <Icon name={option.icon} size={option.iconSize} title={option.label} />}
          {option.label && <span>{option.label}</span>}
        </Radio.Button>
      ))}
    </Radio.Group>
  );
};

export default RadioGroup;
