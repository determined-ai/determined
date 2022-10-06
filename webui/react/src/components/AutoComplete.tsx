import { AutoComplete as AutoCompleteAntD, Input } from 'antd';
import React, { useCallback, useEffect, useState } from 'react';

import css from './AutoComplete.module.scss';

export interface OptionType {
  disabled?: boolean;
  label: string;
  value: string | number;
}

interface Props extends React.ComponentProps<typeof AutoCompleteAntD<string, OptionType>> {
  initialValue?: string;
  onSave?: (option?: OptionType | string) => void;
}

const AutoComplete = ({ initialValue, onSave, ...props }: Props): React.ReactElement => {
  const [value, setValue] = useState(initialValue);

  useEffect(() => {
    setValue(initialValue);
  }, [initialValue]);

  const classes = [css.base];
  if (value === undefined || value.length === 0) classes.push(css.empty);

  const handleSelect = useCallback(
    (value: string, option: OptionType) => {
      setValue(option.label);
      onSave?.(option);
    },
    [onSave],
  );

  const handleBlur = useCallback(() => {
    const selectedOption = props.options?.find((option) => option.label === value) ?? value?.trim();
    onSave?.(selectedOption);
  }, [onSave, props.options, value]);

  const handleClear = useCallback(() => {
    setValue(undefined);
    onSave?.();
  }, [onSave]);

  return (
    <div className={classes.join(' ')}>
      <AutoCompleteAntD<string, OptionType>
        {...props}
        value={value}
        onBlur={handleBlur}
        onChange={setValue}
        onClear={handleClear}
        onSelect={handleSelect}>
        <Input onPressEnter={handleBlur} />
      </AutoCompleteAntD>
    </div>
  );
};

export default AutoComplete;
