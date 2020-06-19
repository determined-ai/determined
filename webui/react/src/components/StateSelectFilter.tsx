import { Select } from 'antd';
import { SelectValue } from 'antd/es/select';
import React, { useCallback } from 'react';

import { CommandState, RunState } from 'types';
import { commandStateToLabel, runStateToLabel } from 'utils/types';

import SelectFilter, { ALL_VALUE } from './SelectFilter';

const { OptGroup, Option } = Select;

interface Props {
  onChange: (value: SelectValue) => void;
  value?: SelectValue;
}

const StateSelectFilter: React.FC<Props> = ({ onChange, value }: Props) => {
  const handleSelect = useCallback((newValue: SelectValue) => {
    const singleValue = Array.isArray(newValue) ? newValue[0] : newValue;
    onChange(singleValue);
  }, [ onChange ]);

  return (
    <SelectFilter label="State" value={value} onSelect={handleSelect}>
      <Option key={ALL_VALUE} value={ALL_VALUE}>All</Option>
      <OptGroup key="expGroup" label="Experiment States">
        {Object.values(RunState).map((value) => (
          <Option key={value} value={value}>{runStateToLabel[value]}</Option>
        ))}
      </OptGroup>
      <OptGroup key="cmdGroup" label="Command States">
        {Object.values(CommandState).map((value) => (
          <Option key={value} value={value}>{commandStateToLabel[value]}</Option>
        ))}
      </OptGroup>
    </SelectFilter>
  );
};

export default StateSelectFilter;
