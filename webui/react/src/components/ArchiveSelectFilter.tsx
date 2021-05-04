import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback } from 'react';

import SelectFilter from 'components/SelectFilter';

const { Option } = Select;

interface Props {
  onChange?: (value: string) => void;
  value?: string;
}

const ArchiveSelectFilter: React.FC<Props> = ({
  onChange,
  value,
}: Props) => {
  const handleSelect = useCallback((newValue: SelectValue) => {
    if (!onChange) return;
    onChange(newValue.toString());
  }, [ onChange ]);
  return (
    <SelectFilter
      dropdownMatchSelectWidth={130}
      label="Show"
      style={{ minWidth: 100 }}
      value={value}
      onSelect={handleSelect}>
      <Option key="All" value="All">All</Option>
      <Option key="Unarchived" value="Unarchived">Unarchived</Option>
      <Option key="Archived" value="Archived">Archived</Option>
    </SelectFilter>
  );
};

export default ArchiveSelectFilter;
