import { AutoComplete as AutoCompleteAntD } from 'antd';
import { DefaultOptionType } from 'antd/lib/select';
import React, { useCallback, useState } from 'react';

import css from './AutoComplete.module.scss';

interface Props<OptionType extends DefaultOptionType>
  extends React.ComponentProps<typeof AutoCompleteAntD<string, OptionType>> {
  onSave?: (option?: OptionType | string) => void;
}

const AutoComplete = <OptionType extends DefaultOptionType>({
  onSave,
  ...props
}: Props<OptionType>): React.ReactElement => {
  const [value, setValue] = useState(props.value ?? '');

  const classes = [css.base];
  if (value.length === 0) classes.push(css.empty);

  const handleSelect = useCallback(
    (value: string, option: OptionType) => {
      onSave?.(option);
    },
    [onSave],
  );

  const handleBlur = useCallback(() => {
    const selectedOption = props.options?.find((option) => option.label === value) ?? value;
    onSave?.(selectedOption);
  }, [onSave, props.options, value]);

  const handleClear = useCallback(() => {
    setValue('');
    onSave?.();
  }, [onSave]);

  return (
    <div className={classes.join(' ')}>
      <AutoCompleteAntD<string, OptionType>
        {...props}
        //className={value.length === 0 ? undefined : css.empty}
        value={value}
        onBlur={handleBlur}
        onChange={setValue}
        onClear={handleClear}
        onSelect={handleSelect}
      />
    </div>
  );
};

export default AutoComplete;
