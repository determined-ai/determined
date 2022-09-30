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
  const [searchValue, setSearchValue] = useState(props.searchValue ?? '');

  const handleSelect = useCallback(
    (value: string, option: OptionType) => {
      onSave?.(option);
    },
    [onSave],
  );

  const handleBlur = useCallback(() => {
    const selectedOption =
      props.options?.find((option) => option.label === searchValue) ?? searchValue;
    onSave?.(selectedOption);
  }, [onSave, props.options, searchValue]);

  const handleClear = useCallback(() => {
    onSave?.();
  }, [onSave]);

  return (
    <div className={css.base}>
      <AutoCompleteAntD<string, OptionType>
        {...props}
        className={searchValue ? undefined : css.empty}
        searchValue={searchValue}
        onBlur={handleBlur}
        onChange={setSearchValue}
        onClear={handleClear}
        onSelect={handleSelect}
      />
    </div>
  );
};

export default AutoComplete;
