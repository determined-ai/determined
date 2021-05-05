import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback } from 'react';

import SelectFilter from 'components/SelectFilter';
import { ALL_VALUE, ArchiveFilters } from 'types';

const { Option } = Select;
interface Props {
  onChange?: (value: ArchiveFilters) => void;
  value?: ArchiveFilters;
}

const archiveOptions = [ ALL_VALUE, 'unarchived', 'archived' ];

const ArchiveSelectFilter: React.FC<Props> = ({
  onChange,
  value,
}: Props) => {
  const handleSelect = useCallback((newValue: SelectValue) => {
    if (!onChange) return;
    const strValue = newValue.toString() as ArchiveFilters;
    if (!(archiveOptions.includes(strValue))) return;
    onChange(strValue);
  }, [ onChange ]);
  return (
    <SelectFilter
      dropdownMatchSelectWidth={130}
      label="Show"
      style={{ minWidth: 100, textTransform: 'capitalize' }}
      value={value || ALL_VALUE}
      onSelect={handleSelect}>
      {archiveOptions.map(option => <Option key={option} value={option}>{option}</Option>)}
    </SelectFilter>
  );
};

export default ArchiveSelectFilter;
